package email

import "testing"

func TestIsMTALeakHeader(t *testing.T) {
	for _, name := range []string{
		"Received", "Return-Path", "User-Agent", "x-virtual-mta", "x-job",
		"X-Originating-IP", "DKIM-Signature",
	} {
		if !IsMTALeakHeader(name) {
			t.Fatalf("want leak: %s", name)
		}
	}
	for _, name := range []string{"From", "To", "Subject", "Message-ID", "X-Mailer", "List-Unsubscribe"} {
		if IsMTALeakHeader(name) {
			t.Fatalf("must keep: %s", name)
		}
	}
}
