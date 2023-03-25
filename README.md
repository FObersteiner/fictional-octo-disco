# fictional-octo-disco
some sensors and a database

## DB
influxDB running on a rpi4.

## data collection and logging
UDP "servers" on the microcontrollers, queried by a `go` application (datalogger). The datalogger uploads the data to the DB and saves it to `csv` files as well.

## data serving and visualization
`dash` app (`Python`) for the browser.
