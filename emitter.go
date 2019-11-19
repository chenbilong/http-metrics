package main

import (
	"io"
	"log"
	"time"
)

// Start initialize loop for sending data from inputs to outputs
func Start(stop chan int) {
	for _, in := range Plugins.Inputs {
		go func(in io.Reader) {
			if err := CopyMulty(in, Plugins.Outputs...); err != nil {
				log.Println("Error during copy: ", err)
				close(stop)
			}
		}(in)
	}

	for _, out := range Plugins.Outputs {
		if r, ok := out.(io.Reader); ok {
			go func(r io.Reader) {
				if err := CopyMulty(r, Plugins.Outputs...); err != nil {
					log.Println("Error during copy: ", err)
					close(stop)
				}
			}(r)
		}
	}

	for {
		select {
		case <-stop:
			finalize()
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// CopyMulty copies from 1 reader to multiple writers
func CopyMulty(src io.Reader, writers ...io.Writer) (err error) {
	buf := make([]byte, Settings.copyBufferSize)
	wIndex := 0
	filteredRequests := make(map[string]time.Time)
	filteredRequestsLastCleanTime := time.Now()

	i := 0

	for {
		nr, er := src.Read(buf)

		if er == io.EOF {
			return nil
		}
		if er != nil {
			return err
		}

		_maxN := nr
		if nr > 500 {
			_maxN = 500
		}
		if nr > 0 && len(buf) > nr {
			payload := buf[:nr]
			meta := payloadMeta(payload)
			if len(meta) < 3 {
				if Settings.debug {
					Debug("[EMITTER] Found malformed record", string(payload[0:_maxN]), nr, "from:", src)
				}
				continue
			}

			if nr >= 5*1024*1024 {
				log.Println("INFO: Large packet... We received ", len(payload), " bytes from ", src)
			}

			if Settings.debug {
				Debug("[EMITTER] input:", string(payload[0:_maxN]), nr, "from:", src)
			}

			if Settings.splitOutput {
				// Simple round robin
				if _, err := writers[wIndex].Write(payload); err != nil {
					return err
				}

				wIndex++

				if wIndex >= len(writers) {
					wIndex = 0
				}
			} else {
				for _, dst := range writers {
					if _, err := dst.Write(payload); err != nil {
						return err
					}
				}
			}
		} else if nr > 0 {
			log.Println("WARN: Packet", nr, "bytes is too large to process. Consider increasing --copy-buffer-size")
		}

		// Run GC on each 1000 request
		if i%1000 == 0 {
			// Clean up filtered requests for which we didn't get a response to filter
			now := time.Now()
			if now.Sub(filteredRequestsLastCleanTime) > 60*time.Second {
				for k, v := range filteredRequests {
					if now.Sub(v) > 60*time.Second {
						delete(filteredRequests, k)
					}
				}
				filteredRequestsLastCleanTime = time.Now()
			}
		}

		i++
	}

	return err
}
