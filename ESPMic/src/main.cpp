#include "FastLED.h"
#include <Arduino.h>
#include <ArduinoJson.h>
#include <AsyncUDP.h>
#include <MicBuffer.h>
#include <WiFi.h>
#include <driver/i2s.h>

// Replace with your Wi-Fi credentials
const char *ssid = "{ YOUR WIFI SSID }";
const char *password = "{ YOUR WIFI PASSWORD }";

// Remote LLM and voice server
IPAddress serverIP(100, 70, 0, 13);
const uint16_t serverPort = 5001;

AsyncUDP udp;

// Indication
#define NUM_LEDS 5
#define LED_TYPE WS2812
#define COLOR_ORDER GRB
#define DATA_PIN GPIO_NUM_22

CRGBArray<NUM_LEDS> leds;

// LED pin
const int LED_PIN = 2;

// Speech detection parameters
const int AUDIO_LOOKBACK = 5; // how many previous samples to send to the server
                              // when speech is detected
const int AUDIO_LISTENING_WINDOW_MS =
    750; // how long to listen for speech after the last speech detection
const int SAMPLE_RATE = 16000;
const int ENERGY_THRESHOLD = 1000000;
const int ZCR_THRESHOLD = 100;

// Microphone options

#define I2S_MIC_CHANNEL I2S_CHANNEL_FMT_ONLY_LEFT
#define I2S_MIC_SERIAL_CLOCK GPIO_NUM_14
#define I2S_MIC_LEFT_RIGHT_CLOCK GPIO_NUM_15
#define I2S_MIC_SERIAL_DATA GPIO_NUM_32

MicBuffer mic(i2s_pin_config_t{.bck_io_num = I2S_MIC_SERIAL_CLOCK,
                               .ws_io_num = I2S_MIC_LEFT_RIGHT_CLOCK,
                               .data_out_num = I2S_PIN_NO_CHANGE,
                               .data_in_num = I2S_MIC_SERIAL_DATA},
              SAMPLE_RATE, ENERGY_THRESHOLD, ZCR_THRESHOLD);

// Payloads
DynamicJsonDocument command(1024);

void setup() {
  Serial.begin(115200);
  Serial.println("\nConnecting to Wi-Fi...");

  FastLED.addLeds<NEOPIXEL, DATA_PIN>(leds, NUM_LEDS);

  leds[0] = CRGB::Black;
  leds[1] = CRGB::Black;
  leds[2] = CRGB::Black;
  leds[3] = CRGB::Black;
  leds[4] = CRGB::Black;
  FastLED.show();

  // Now turn the LED off, then pause

  WiFi.begin(ssid, password);

  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }

  Serial.println("\nWi-Fi connected.");
  Serial.print("IP address: ");
  Serial.println(WiFi.localIP());

  pinMode(LED_PIN, OUTPUT);
  digitalWrite(LED_PIN, LOW);

  // start up the I2S peripheral
  mic.begin();

  // Connect to remote server
  udp.connect(serverIP, serverPort);

  udp.onPacket([](AsyncUDPPacket packet) {
    Serial.print("Got command from server: ");
    Serial.write(packet.data(), packet.length());
    Serial.println();

    // unpacking
    deserializeJson(command, packet.data());

    // iterate over 5 leds
    for (int i = 0; i < 5; i++) {
      byte red = command[i]["red"];
      byte green = command[i]["green"];
      byte blue = command[i]["blue"];

      leds[i] = CRGB(red, green, blue);
    }

    FastLED.show();
  });
}

unsigned long lastSpeechDetected = 0;
void loop() {
  mic.update();

  if (mic.isSpeech()) {
    if (millis() - lastSpeechDetected > AUDIO_LISTENING_WINDOW_MS) {
      // to avoid audio chopping we will send AUDIO_LOOKBACK previous samples to
      // the server
      for (int i = 1; i < AUDIO_LOOKBACK; i++) {
        AsyncUDPMessage msg = AsyncUDPMessage();
        msg.write((uint8_t *)mic.getPreviousSample(i),
                  SAMPLE_BUFFER_SIZE * sizeof(int16_t));
        udp.send(msg);
      }
    }

    lastSpeechDetected = millis();
    digitalWrite(LED_PIN, HIGH);
  }

  if (millis() - lastSpeechDetected > AUDIO_LISTENING_WINDOW_MS) {
    digitalWrite(LED_PIN, LOW);
  } else if (udp.connected()) {
    AsyncUDPMessage msg = AsyncUDPMessage();
    msg.write((uint8_t *)mic.getLastSample(),
              SAMPLE_BUFFER_SIZE * sizeof(int16_t));
    udp.send(msg);
  }
}