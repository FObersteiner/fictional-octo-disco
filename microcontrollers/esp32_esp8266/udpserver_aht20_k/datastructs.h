// SensorData - a structure to hold sensor data; make it global we don't have to
// pass pointers around.
typedef struct {
  // humidity / aht20
  double T_aht20;
  double rH;
  double aH; // A [g / m³]
//  // pressure / lps22
//  double T_lps22;
//  double p;
} SensorData;
