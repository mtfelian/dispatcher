package dispatcher_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("checks dispatcher, OK", func() {
		totalExpected, urlsSent := 0, 500

		for i := 0; i < urlsSent; i++ {
			r := rand.Intn(30)
			totalExpected += r

			go func() {
				defer GinkgoRecover()
				url := fmt.Sprintf("%s/treat", server.URL)
				b, err := json.Marshal(testInput{
					URL:  fmt.Sprintf("%s%s%d", mServer.URL, pageAddr, r),
					What: "Go",
				})
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.Post(url, "application/json", bytes.NewReader(b))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			}()

			// check that workers count don't exceeds max limit
			workers := dispatcher.Workers()
			fmt.Println("Workers:", workers)
		}

		attempts, ticker := 0, time.NewTicker(time.Second)
		for {
			<-ticker.C
			attempts++
			if dispatcher.TasksDone() == urlsSent {
				ticker.Stop()
				break
			}
			Expect(attempts).To(BeNumerically("<=", 5), "too long!")
		}

		Expect(dispatcher.ErrorCount()).To(Equal(0))

		Expect(totalFound.Get()).To(BeNumerically("==", totalExpected),
			"totalFound != totalExpected, %d != %d", totalFound, totalExpected)

		dispatcher.Stop()
	})
})
