#include "MicBuffer.h"

MicBuffer::MicBuffer(i2s_pin_config_t pin_config, int sample_rate,
                     int energy_threshold, int zcr_threshold)
    : _pin_config(pin_config), _sample_rate(sample_rate),
      _energy_threshold(energy_threshold), _zcr_threshold(zcr_threshold),
      _circular_buffer_pos(0), _circular_buffer_oldest(0) {
  memset(_circular_buffer, 0,
         CIRCULAR_BUFFER_SIZE * SAMPLE_BUFFER_SIZE * sizeof(int16_t));
}

void MicBuffer::begin() {
  i2s_config_t i2s_config = {
      .mode = (i2s_mode_t)(I2S_MODE_MASTER | I2S_MODE_RX),
      .sample_rate = _sample_rate,
      .bits_per_sample = I2S_BITS_PER_SAMPLE_16BIT,
      .channel_format = I2S_CHANNEL_FMT_ONLY_LEFT,
      .communication_format =
          i2s_comm_format_t(I2S_COMM_FORMAT_I2S | I2S_COMM_FORMAT_I2S_MSB),
      .intr_alloc_flags = ESP_INTR_FLAG_LEVEL1,
      .dma_buf_count = 2,
      .dma_buf_len = sizeof(_sample_buffer),
  };

  i2s_driver_install(I2S_NUM_0, &i2s_config, 0, NULL);
  i2s_set_pin(I2S_NUM_0, &_pin_config);
}

void MicBuffer::update() {
  size_t bytes_read = 0;

  _circular_buffer_pos = (_circular_buffer_pos + 1) % CIRCULAR_BUFFER_SIZE;
  if (_circular_buffer_pos == _circular_buffer_oldest) {
    _circular_buffer_oldest =
        (_circular_buffer_oldest + 1) % CIRCULAR_BUFFER_SIZE;
  }

  // Read new samples directly into the next position in the circular buffer
  i2s_read(I2S_NUM_0,
           (void *)&_circular_buffer[_circular_buffer_pos * SAMPLE_BUFFER_SIZE],
           SAMPLE_BUFFER_SIZE * sizeof(int16_t), &bytes_read, portMAX_DELAY);

  // Copy the new samples from the circular buffer to _sample_buffer
  memcpy(_sample_buffer,
         &_circular_buffer[_circular_buffer_pos * SAMPLE_BUFFER_SIZE],
         SAMPLE_BUFFER_SIZE * sizeof(int16_t));
}

int16_t *MicBuffer::getPreviousSample(int n) {
  int index;

  // Calculate the index in the circular buffer
  index = (_circular_buffer_pos + CIRCULAR_BUFFER_SIZE - n - 1) %
          CIRCULAR_BUFFER_SIZE;

  // Return the pointer to the requested previous sample
  return &_circular_buffer[index * SAMPLE_BUFFER_SIZE];
}

bool MicBuffer::isSpeech() {
  int64_t energy = 0;
  int zcr = 0;
  int size = sizeof(_sample_buffer) / sizeof(int16_t);

  for (int i = 0; i < size; i++) {
    energy += (int64_t)_sample_buffer[i] * _sample_buffer[i];

    if (i > 0 && ((_sample_buffer[i] >= 0 && _sample_buffer[i - 1] < 0) ||
                  (_sample_buffer[i] < 0 && _sample_buffer[i - 1] >= 0))) {
      zcr++;
    }
  }

  return energy > _energy_threshold && zcr < _zcr_threshold;
}

int16_t *MicBuffer::getLastSample() { return _sample_buffer; }

void MicBuffer::setEnergyThreshold(int energy_threshold) {
  _energy_threshold = energy_threshold;
}

void MicBuffer::setZCRThreshold(int zcr_threshold) {
  _zcr_threshold = zcr_threshold;
}
