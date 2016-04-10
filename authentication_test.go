package main_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/salemove/github-review-helper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = TestWebhookHandler(func(context WebhookTestContext) {
	Describe("authentication", func() {
		var (
			handle      = context.Handle
			headers     = context.Headers
			requestJSON = context.RequestJSON

			responseRecorder *httptest.ResponseRecorder
			pullRequests     *MockPullRequests
		)
		BeforeEach(func() {
			responseRecorder = *context.ResponseRecorder
			pullRequests = *context.PullRequests
		})

		Context("with an empty X-Hub-Signature header", func() {
			headers.Is(func() map[string][]string {
				return map[string][]string{
					"X-Hub-Signature": []string{""},
				}
			})
			It("fails with StatusUnauthorized", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("with an invalid X-Hub-Signature header", func() {
			requestJSON.Is(func() string {
				return "{}"
			})
			headers.Is(func() map[string][]string {
				return map[string][]string{
					"X-Hub-Signature": []string{"sha1=2f539a59127d552f4565b1a114ec8f4fa2d55f55"},
				}
			})

			It("fails with StatusForbidden", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusForbidden))
			})
		})

		Context("with an empty request with a proper signature", func() {
			var validSignature = "sha1=33c829a9c355e7722cb74d25dfa54c6c623cde63"
			requestJSON.Is(func() string {
				return "{}"
			})
			headers.Is(func() map[string][]string {
				return map[string][]string{
					"X-Hub-Signature": []string{validSignature},
				}
			})

			It("succeeds with 'ignored' response", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
			})

			Context("with a gibberish event", func() {
				headers.Is(func() map[string][]string {
					return map[string][]string{
						"X-Hub-Signature": []string{validSignature},
						"X-Github-Event":  []string{"gibberish"},
					}
				})

				It("succeeds with 'ignored' response", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
				})
			})
		})

		Context("with a valid signature", func() {
			Describe("issue_comment event", func() {
				headers.Is(func() map[string][]string {
					return map[string][]string{
						"X-Github-Event": []string{"issue_comment"},
					}
				})

				Context("with an arbitrary comment", func() {
					requestJSON.Is(func() string {
						return IssueCommentEvent("just a simple comment")
					})

					It("succeeds with 'ignored' response", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
					})
				})
			})
		})
	})
})
