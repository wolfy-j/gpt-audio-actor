#ifndef AUDIO_PROCESSOR_H
#define AUDIO_PROCESSOR_H

#include <Arduino.h>
#include <driver/i2s.h>

#define SAMPLE_BUFFER_SIZE 512
#define CIRCULAR_BUFFER_SIZE 10

class MicBuffer {
public:
  MicBuffer(i2s_pin_config_t pin_config, int sample_rate, int energy_threshold,
            int zcr_threshold);
  void begin();
  void update();
  bool isSpeech();
  int16_t *getLastSample();
  int16_t *getPreviousSample(int n);
  void setEnergyThreshold(int energy_threshold);
  void setZCRThreshold(int zcr_threshold);

private:
  i2s_pin_config_t _pin_config;
  int _sample_rate;
  int _energy_threshold;
  int _zcr_threshold;

  // buffers
  int16_t _sample_buffer[SAMPLE_BUFFER_SIZE];
  int16_t _circular_buffer[CIRCULAR_BUFFER_SIZE * SAMPLE_BUFFER_SIZE];
  int _circular_buffer_pos;
  int _circular_buffer_oldest;
};

#endif // AUDIO_PROCESSOR_H
