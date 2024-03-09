#include "WiFi.h"
#include "arduino_secrets.h"
#include <WiFiUdp.h>
WiFiUDP Udp;

// NodeMCU-32S

#include "datastructs.h"
#include "DHT.h"
#define DHTPIN 4     // Digital pin connected to the DHT sensor
#define DHTTYPE DHT22   // DHT 22  (AM2302), AM2321
DHT dht(DHTPIN, DHTTYPE);

int status = WL_DISCONNECTED;
char ssid[] = SECRET_SSID; // login info from arduino_secrets.h
char pass[] = SECRET_PASS;

int last_ip_octet = 112; // Draussen
IPAddress ip(192, 168, 178, last_ip_octet); // fix IP address
IPAddress gateway(192, 168, 178, 1);
IPAddress subnet(255, 255, 255, 0);

unsigned int udpPort = 16083; // port for UDP communication

char packetBuffer[128]; // buffer to hold incoming packet
char replyBuffer[128]; // buffer for the string to send back


void setup() {
  Serial.begin(115200);

  Serial.println("Connecting to DHT22 sensor");
  dht.begin();

  // ~~~~~ WIFI ~~~~~
  // Configures static IP address
  if (!WiFi.config(ip, gateway, subnet)) {
    Serial.println("STA Failed to configure");
  }
  WiFi.begin(ssid, pass);
    // attempt to connect to Wifi network:
  while (WiFi.status() != WL_CONNECTED) {
    Serial.print(".");
    // wait 1 second for re-trying
    delay(1000);
  }

  Serial.print("Connected to ");
  Serial.println(ssid);
  printWifiStatus();

  Udp.begin(udpPort);
  Serial.println("UDP server ready...");
}

// void loop() {
//   // Wait a few seconds between measurements.
//   delay(2000);

//   // Reading temperature or humidity takes about 250 milliseconds!
//   // Sensor readings may also be up to 2 seconds 'old' (its a very slow sensor)
//   float h = dht.readHumidity();
//   // Read temperature as Celsius (the default)
//   float t = dht.readTemperature();
//   // Read temperature as Fahrenheit (isFahrenheit = true)
//   float f = dht.readTemperature(true);

//   // Check if any reads failed and exit early (to try again).
//   if (isnan(h) || isnan(t) || isnan(f)) {
//     Serial.println(F("Failed to read from DHT sensor!"));
//     return;
//   }

//   // Compute heat index in Fahrenheit (the default)
//   float hif = dht.computeHeatIndex(f, h);
//   // Compute heat index in Celsius (isFahreheit = false)
//   float hic = dht.computeHeatIndex(t, h, false);

//   Serial.print(F("Humidity: "));
//   Serial.print(h);
//   Serial.print(F("%  Temperature: "));
//   Serial.print(t);
//   Serial.print(F(" C "));
//   Serial.print(f);
//   Serial.print(F(" F  Heat index: "));
//   Serial.print(hic);
//   Serial.print(F(" C "));
//   Serial.print(hif);
//   Serial.println(F(" F"));
// }

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

    refreshSensorData();
    json = makeJSON();

    Serial.println("Reply:");
    Serial.println(json);
    jsonLen = json.length();
    // clear replyBuffer
    for ( int i = 0; i < sizeof(replyBuffer);  ++i )
      replyBuffer[i] = (char)0;
    // fill replyBuffer with JSON
    for (int i = 0; i < jsonLen; i++ ) {
      replyBuffer[i] = json[i];
    }

    // send a reply, to the IP address and port that sent us the packet we received
    Udp.beginPacket(Udp.remoteIP(), Udp.remotePort());
    Udp.print(replyBuffer);
    Udp.endPacket();
  }

  delay(100);

}


// refreshSensorData - a function to refresh content of the sensor data structure
void refreshSensorData() {

  s.T_dht22 = dht.readTemperature();
  s.rH = dht.readHumidity();

  // calculate abs. humidity based on Magnus-Tetens equation
  s.aH = (6.1094 * exp((17.625 * s.T_dht22) / (s.T_dht22 + 243.5)) * s.rH * 2.1674) / (273.15 + s.T_dht22);

}


// makeJSON - make a json string from sensor data
String makeJSON() {
  String out = "{\"ID\": " + String(last_ip_octet) + ", ";
  out += "\"T\": " + String(s.T_dht22, 5) + ", ";
  out += "\"rH\": " + String(s.rH, 5) + ", ";
  out += "\"aH\": " + String(s.aH, 5) + "}";
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
