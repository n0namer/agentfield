package server

import "testing"

func TestHTTPListenAddressLocalOnlyOptIn(t *testing.T) {
 t.Setenv("AGENTFIELD_BIND_LOOPBACK_ONLY", "")
 if got := httpListenAddress(18080); got != ":18080" {
  t.Fatalf("default changed: %s", got)
 }
 t.Setenv("AGENTFIELD_BIND_LOOPBACK_ONLY", "1")
 if got := httpListenAddress(18080); got != "127.0.0.1:18080" {
  t.Fatalf("not loopback: %s", got)
 }
}
