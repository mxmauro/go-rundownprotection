package main

import (
	"log"
	"sync"
	"time"

	rp "github.com/randlabs/rundown-protection"
)

func main() {
	var wg sync.WaitGroup

	//create the rundown protection
	r := rp.Create()

	//run 5 goroutines which will complete after 5 seconds
	for i := 1; i <= 5; i++ {
		if r.Acquire() {
			log.Printf("Starting goroutine #%v.", i)

			go func(idx int) {
				time.Sleep(5 * time.Second)

				log.Printf("Goroutine #%v completed.", idx)
				r.Release()
			}(i)
		} else {
			log.Printf("Unable to start goroutine #%v.", i)
		}
	}

	//the following goroutine will run after 10 seconds simulating acquisition of the rundown object
	//after the wait call
	wg.Add(1)
	go func() {
		//wait 10 seconds
		time.Sleep(10 * time.Second)

		//run a new set of goroutines
		for i := 6; i <= 10; i++ {
			if r.Acquire() {
				log.Printf("Starting goroutine #%v.", i)
				go func(idx int) {
					time.Sleep(5 * time.Second)

					log.Printf("Goroutine #%v completed.", idx)
					r.Release()
				}(i)
			} else {
				log.Printf("Unable to start goroutine #%v.", i)
			}
		}

		wg.Done()
	}()

	log.Printf("Before rundown wait")
	r.Wait()
	log.Printf("After rundown wait")

	wg.Wait()
	log.Printf("Done!")
}
