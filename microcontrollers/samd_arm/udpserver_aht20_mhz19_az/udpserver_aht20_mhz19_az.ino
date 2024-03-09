// wifi
#include <SPI.h>
#include <WiFiNINA.h>
#include <WiFiUdp.h>
#include "arduino_secrets.h"
 
// sensors
#include <Wire.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_Sensor.h>
#include "datastructs.h"
#include "wiring_private.h"
#include "MHZ19.h"  

Uart mySerial (&sercom0, 5, 6, SERCOM_RX_PAD_1, UART_TX_PAD_0);
// Attach the interrupt handler to the SERCOM
void SERCOM0_Handler()
{
    mySerial.IrqHandler();
}

MHZ19 myMHZ19;
Adafruit_AHTX0 aht;

int status = WL_DISCONNECTED;
char ssid[] = SECRET_SSID; 
char pass[] = SECRET_PASS;

int last_ip_octet = 109; // AZ
IPAddress ip(192, 168, 178, last_ip_octet);  // fix IP address
unsigned int udpPort = 16083;              // port for UDP communication

char packetBuffer[128];  // buffer to hold incoming packet
char replyBuffer[128];   // buffer for the string to send back
WiFiUDP Udp;


// setup checks that the sensors are there and connects to specified WiFi
void setup() {
  // alternative serial com
  pinPeripheral(5, PIO_SERCOM_ALT);
  pinPeripheral(6, PIO_SERCOM_ALT);
  // pinMode(LED_BUILTIN, OUTPUT);
  // digitalWrite(LED_BUILTIN, LOW);

  // ~~~~~ Serial Coms and Sensors ~~~~~
  mySerial.begin(9600);
  myMHZ19.begin(mySerial);

  Serial.begin(115200);

  //  // remove for testing without USB:
  //  while (!Serial) {
  //    ; // wait for serial port to connect. Needed for native USB port only
  //  }

  Serial.println("Connecting to AHT20 sensor");
  if (!aht.begin()) {
    Serial.println("Could not find AHT - Check wiring");
    while (1) delay(1000);
  }
  Serial.println("AHT20 found");

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
  WiFi.setHostname("InsideArduino");

  while (status != WL_CONNECTED) {
    Serial.print("Connecting to SSID: ");
    Serial.println(ssid);
    status = WiFi.begin(ssid, pass);
    delay(1000);
  }

  Serial.println("Connected to WiFi");
  printWifiStatus();

  Udp.begin(udpPort);
  Serial.println("UDP server ready...");
  // digitalWrite(LED_BUILTIN, HIGH);
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

    // digitalWrite(LED_BUILTIN, LOW);
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
    // digitalWrite(LED_BUILTIN, HIGH);
  }

  delay(100);
}


// refreshSensorData - a function to refresh content of the sensor data structure
void refreshSensorData() {

  sensors_event_t humidity, temp1;
  aht.getEvent(&humidity, &temp1);

  s.T_aht20 = temp1.temperature;
  s.rH = humidity.relative_humidity;

  // calculate abs. humidity based on Magnus-Tetens equation
  s.aH = (6.1094 * exp((17.625 * s.T_aht20) / (s.T_aht20 + 243.5)) * s.rH * 2.1674) / (273.15 + s.T_aht20);

  s.CO2 = myMHZ19.getCO2();
}


// makeJSON - make a json string from sensor data
String makeJSON() {
  String out = "{\"ID\": " + String(last_ip_octet) + ", ";
  out += "\"T\": " + String(s.T_aht20, 5) + ", ";
  out += "\"rH\": " + String(s.rH, 5) + ", ";
  out += "\"aH\": " + String(s.aH, 5) + ", ";
  out += "\"CO2\": " + String(s.CO2) + "}";
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
