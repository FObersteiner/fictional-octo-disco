# fictional-octo-disco
Some sensors and a database. The name is just the best that the github auto-generator came up with (imho)...

## DB
`influxDB` running on a rpi4.

## data collection and logging
UDP servers on the microcontrollers, queried by a `go` application (datalogger). The datalogger uploads the data to the DB and saves it to `csv` files as well.

## data serving and visualization
`dash` app (`Python`) for the browser.
