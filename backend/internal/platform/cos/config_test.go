package cos

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("COS_BUCKET", "")
	t.Setenv("COS_REGION", "")
	t.Setenv("COS_SECRET_ID", "")
	t.Setenv("COS_SECRET_KEY", "")

	cfg := LoadConfig()
	if cfg.Bucket != defaultBucket || cfg.Region != defaultRegion {
		t.Fatalf("defaults: bucket=%q region=%q", cfg.Bucket, cfg.Region)
	}
	if cfg.IsConfigured() {
		t.Fatal("expected not configured without secrets")
	}
}
