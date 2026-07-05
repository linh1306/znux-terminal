package specs

import "testing"

func TestParseSSListeningPorts(t *testing.T) {
	out := []byte(`tcp LISTEN 0 4096 127.0.0.1:5432 0.0.0.0:* users:(("postgres",pid=1234,fd=7))
udp UNCONN 0 0 0.0.0.0:5353 0.0.0.0:* users:(("mdns",pid=55,fd=4))
tcp LISTEN 0 4096 [::1]:8080 [::]:* users:(("node",pid=999,fd=21))
tcp LISTEN 0 4096 127.0.0.53%lo:53 0.0.0.0:* users:(("systemd-resolve",pid=777,fd=17))
tcp LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:(("next-server (v1",pid=484781,fd=21))
`)

	got := parseSSListeningPorts(out, []string{"tcp", "udp"})

	if len(got) != 5 {
		t.Fatalf("ports len = %d, want 5: %#v", len(got), got)
	}
	if got[0].Port != "5432" || got[0].Protocol != "tcp" {
		t.Fatalf("unexpected first port: %#v", got[0])
	}
	if got[0].Description() != "LISTEN 127.0.0.1 postgres pid=1234" {
		t.Fatalf("first description = %q", got[0].Description())
	}
	if got[1].Port != "5353" || got[1].Protocol != "udp" {
		t.Fatalf("unexpected second port: %#v", got[1])
	}
	if got[2].Port != "8080" || got[2].Protocol != "tcp" {
		t.Fatalf("unexpected third port: %#v", got[2])
	}
	if got[2].Description() != "LISTEN ::1 node pid=999" {
		t.Fatalf("third description = %q", got[2].Description())
	}
	if got[3].Description() != "LISTEN 127.0.0.53 systemd-resolve pid=777" {
		t.Fatalf("fourth description = %q", got[3].Description())
	}
	if got[4].Description() != "LISTEN 127.0.0.1 next-server pid=484781" {
		t.Fatalf("fifth description = %q", got[4].Description())
	}
}

func TestSortListeningPortsUsesProtocolOrder(t *testing.T) {
	ports := []listeningPort{
		{Protocol: "udp", Port: "53"},
		{Protocol: "tcp", Port: "3000"},
		{Protocol: "tcp", Port: "22"},
	}

	sortListeningPorts(ports, []string{"tcp", "udp"})

	if ports[0].Protocol != "tcp" || ports[0].Port != "22" {
		t.Fatalf("first port = %#v, want tcp 22", ports[0])
	}
	if ports[1].Protocol != "tcp" || ports[1].Port != "3000" {
		t.Fatalf("second port = %#v, want tcp 3000", ports[1])
	}
	if ports[2].Protocol != "udp" || ports[2].Port != "53" {
		t.Fatalf("third port = %#v, want udp 53", ports[2])
	}
}
