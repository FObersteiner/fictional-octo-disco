// SensorData - a structure to hold sensor data; make it global we don't have to
// pass pointers around.
typedef struct {
  double T_dht22;
  double rH;
  double aH;
} SensorData;
