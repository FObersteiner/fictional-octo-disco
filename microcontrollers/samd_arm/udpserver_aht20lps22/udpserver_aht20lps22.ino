
// wifi
#include <SPI.h>
#include <WiFiNINA.h>
#include <WiFiUdp.h>
#include "arduino_secrets.h"
// sensors
#include <Wire.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_LPS2X.h>
#include <Adafruit_Sensor.h>
#include "datastructs.h"

Adafruit_AHTX0 aht;
Adafruit_LPS22 lps;

int status = WL_IDLE_STATUS;
char ssid[] = SECRET_SSID;  // login info from arduino_secrets.h
char pass[] = SECRET_PASS;

int last_ip_octet = 58;
IPAddress ip(192, 168, 0, last_ip_octet);  // fix IP address
unsigned int udpPort = 16083;              // port for UDP communication

char packetBuffer[128];  // buffer to hold incoming packet
char replyBuffer[128];   // buffer for the string to send back
WiFiUDP Udp;


// setup checks that the sensors are there and connects to specified WiFi
void setup() {
  pinMode(LED_BUILTIN, OUTPUT);
  digitalWrite(LED_BUILTIN, LOW);
  // ~~~~~ Serial Coms and Sensors ~~~~~
  Serial.begin(115200);

  //  // remove for testing without USB:
  //  while (!Serial) {
  //    ; // wait for serial port to connect. Needed for native USB port only
  //  }

  Serial.println("Connecting to AHT20 + LPS22 sensors");
  if (!aht.begin()) {
    Serial.println("Could not find AHT - Check wiring");
    while (1) delay(10);
  }
  Serial.println("AHT20 found");

  if (!lps.begin_I2C()) {
    Serial.println("Could not find LPS - Check wiring");
    while (1) {
      delay(10);
    }
  }
  Serial.println("LPS22 found");
  lps.setDataRate(LPS22_RATE_1_HZ);

  // ~~~~~ WIFI ~~~~~
  if (WiFi.status() == WL_NO_MODULE) {
    Serial.println("Communication with WiFi module failed!");
    // don't continue
    while (true)
      ;
  }

  String fv = WiFi.firmwareVersion();
  if (fv < WIFI_FIRMWARE_LATEST_VERSION) {
    Serial.println("Please upgrade the firmware");
  }

  // set IP and attempt to connect to WiFi network:
  WiFi.config(ip);
  WiFi.setHostname("OutsideArduino");

  while (status != WL_CONNECTED) {
    Serial.print("Connecting to SSID: ");
    Serial.println(ssid);
    status = WiFi.begin(ssid, pass);
    delay(10000);
  }

  Serial.println("Connected to WiFi");
  printWifiStatus();

  Udp.begin(udpPort);
  Serial.println("\nUDP server ready...");
  digitalWrite(LED_BUILTIN, HIGH);
}


SensorData s;
String json;
int jsonLen;

void loop() {
  status = WiFi.status();
  if (status != WL_CONNECTED) {
    Serial.println("connection lost, falling back to setup...");
    setup();
  }

  // if there's data available, read a packet
  int packetSize = Udp.parsePacket();
  if (packetSize) {
    Serial.print("Received packet of size ");
    Serial.println(packetSize);
    Serial.print("From ");
    IPAddress remoteIp = Udp.remoteIP();
    Serial.print(remoteIp);
    Serial.print(", port ");
    Serial.println(Udp.remotePort());

    // read the packet into packetBufffer
    int len = Udp.read(packetBuffer, 127);
    if (len > 0) {
      packetBuffer[len] = 0;
    }
    Serial.println("Contents:");
    Serial.println(packetBuffer);

    digitalWrite(LED_BUILTIN, LOW);
    refreshSensorData();
    json = makeJSON();

    Serial.println("Reply:");
    Serial.println(json);
    jsonLen = json.length();
    // clear replyBuffer
    for (int i = 0; i < sizeof(replyBuffer); ++i)
      replyBuffer[i] = (char)0;
    // fill replyBuffer with JSON
    for (int i = 0; i < jsonLen; i++) {
      replyBuffer[i] = json[i];
    }

    // send a reply, to the IP address and port that sent us the packet we received
    Udp.beginPacket(Udp.remoteIP(), Udp.remotePort());
    Udp.write(replyBuffer);
    Udp.endPacket();
    digitalWrite(LED_BUILTIN, HIGH);
  }

  delay(100);
}


// refreshSensorData - a function to refresh content of the sensor data structure
void refreshSensorData() {
  // rH sensor
  sensors_event_t humidity, temp1;
  aht.getEvent(&humidity, &temp1);
  // p sensor
  sensors_event_t pressure, temp2;
  lps.getEvent(&pressure, &temp2);

  s.T_aht20 = temp1.temperature;
  s.rH = humidity.relative_humidity;

  s.T_lps22 = temp2.temperature;
  s.p = pressure.pressure;

  // calculate abs. humidity based on Magnus-Tetens equation
  s.aH = (6.1094 * exp((17.625 * s.T_aht20) / (s.T_aht20 + 243.5)) * s.rH * 2.1674) / (273.15 + s.T_aht20);
}


// makeJSON - make a json string from sensor data
String makeJSON() {
  String out = "{\"ID\": " + String(last_ip_octet) + ", ";
  out += "\"T\": " + String(s.T_aht20, 5) + ", ";
  out += "\"rH\": " + String(s.rH, 5) + ", ";
  out += "\"aH\": " + String(s.aH, 5) + ", ";
  out += "\"p\": " + String(s.p, 5) + "}";
  return out;
}

void printWifiStatus() {
  Serial.print("SSID: ");
  Serial.println(WiFi.SSID());

  IPAddress ip = WiFi.localIP();
  Serial.print("IP Address: ");
  Serial.println(ip);

  long rssi = WiFi.RSSI();
  Serial.print("signal strength (RSSI):");
  Serial.print(rssi);
  Serial.println(" dBm");
}
