package main

import (
	"context"
	"net"
	"time"
)

// dataCollector pings Arduinos and forwards data if received
func dataCollector(ctx context.Context, devices []arduino, data chan<- []byte, sigDone chan<- struct{}) {
	var buf = make([]byte, 512)

	// set up UDP socket
	// actually, this is only necessary if we want to send from a specific address/port
	laddr, err := net.ResolveUDPAddr("udp", "")
	if err != nil {
		log.Error().Err(err)
		return
	}
	log.Debug().Msgf("source addr: %v", laddr)

	ticker := time.NewTicker(CHECKINTERVAL)
	defer ticker.Stop()
	for { // outer loop: repeatedly query data from devices unless context is cancelled
		select {
		case <-ctx.Done():
			log.Debug().Msg("data collector ctx: Done !")
			sigDone <- struct{}{}
			log.Debug().Msg("data collector closing")
			return
		case <-ticker.C:
			log.Debug().Msg("check devices...")
			for i, dev := range devices {
				log.Debug().Msgf("check %v", dev.name)
				if dev.last_contact.After(time.Now().Add(-INTERVAL + CHECKINTERVAL)) {
					// if last contact was within INTERVAL, we can just continue
					continue
				}
				log.Debug().Msgf("query %v", dev.name)
				conn, err := net.DialUDP("udp", laddr, dev.UDPaddress)
				if err != nil {
					log.Error().Err(err)
					continue // skip to next device if we cannot dial
				}

				err = conn.SetReadDeadline(time.Now().Add(time.Second))
				if err != nil {
					log.Error().Err(err)
				}

				n, err := conn.Write([]byte("hello"))
				if err != nil {
					log.Error().Err(err)
					continue // skip to next device if we cannot send
				}
				log.Debug().Msgf("wrote %v bytes to %v", n, dev.address)

				// now read
				n, err = conn.Read(buf)
				if err == nil {
					log.Debug().Msgf("received %v", string(buf[:n]))
					log.Debug().Msg("forwarding bytes...")
					// update last_contact
					devices[i].last_contact = time.Now()
					// forward data
					data <- buf[:n]
					continue
				} else {
					log.Error().Err(err)
				}
				// if we reach this point, conn.Read did not return anything
				log.Error().Msgf("no response from %v, %v bytes received", dev.name, n)
			}
		}
	}
}
