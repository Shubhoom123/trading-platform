package config

import "testing"

func TestLoadRejectsWeakSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "too-short")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a sub-32-byte JWT secret")
	}
}

func TestLoadDefaultsAndBrokerParsing(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret-unit-test-secret-0123456789")
	t.Setenv("KAFKA_BOOTSTRAP_SERVERS", "a:9092, b:9092 ,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":8090" {
		t.Fatalf("default listen addr = %q", cfg.ListenAddr)
	}
	if cfg.FillsTopic != "fills" {
		t.Fatalf("default fills topic = %q", cfg.FillsTopic)
	}
	if len(cfg.KafkaBrokers) != 2 ||
		cfg.KafkaBrokers[0] != "a:9092" || cfg.KafkaBrokers[1] != "b:9092" {
		t.Fatalf("broker parsing = %v", cfg.KafkaBrokers)
	}
}

func TestSplitCSVTrimsAndDropsEmpties(t *testing.T) {
	got := splitCSV(" a , , b ,c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
