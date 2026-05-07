package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openlibrecommunity/olcrtc/internal/app/session"
	"github.com/openlibrecommunity/olcrtc/internal/logger"
)

func TestToSessionConfig(t *testing.T) {
	cfg := config{
		mode:            "cnc",
		link:            "direct",
		transport:       "vp8channel",
		carrier:         "jazz",
		roomID:          "room",
		clientID:        "client",
		keyHex:          "key",
		socksHost:       "127.0.0.1",
		socksPort:       1080,
		dnsServer:       "1.1.1.1:53",
		socksProxyAddr:  "proxy",
		socksProxyPort:  1081,
		videoWidth:      640,
		videoHeight:     480,
		videoFPS:        30,
		videoBitrate:    "1M",
		videoHW:         "none",
		videoQRSize:     4,
		videoQRRecovery: "low",
		videoCodec:      "qrcode",
		videoTileModule: 4,
		videoTileRS:     20,
		vp8FPS:          25,
		vp8BatchSize:    8,
		seiFPS:          40,
		seiBatchSize:    3,
		seiFragmentSize: 512,
		seiAckTimeoutMS: 1500,
		lifetime:        300,
	}

	got := toSessionConfig(cfg)
	if got.Mode != cfg.mode || got.Carrier != "jazz" || got.SOCKSPort != cfg.socksPort ||
		got.VideoTileRS != cfg.videoTileRS || got.VP8BatchSize != cfg.vp8BatchSize ||
		got.SEIFPS != cfg.seiFPS || got.SEIBatchSize != cfg.seiBatchSize ||
		got.SEIFragmentSize != cfg.seiFragmentSize || got.SEIAckTimeoutMS != cfg.seiAckTimeoutMS ||
		got.Lifetime != cfg.lifetime {
		t.Fatalf("toSessionConfig() = %+v", got)
	}

}

func TestParseFlagsFrom(t *testing.T) {
	cfg, err := parseFlagsFrom([]string{
		"-mode", "srv",
		"-link", "direct",
		"-transport", "vp8channel",
		"-carrier", "telemost",
		"-id", "room",
		"-client-id", "client",
		"-socks-port", "1080",
		"-socks-host", "127.0.0.1",
		"-key", "key",
		"-debug",
		"-data", "data",
		"-dns", "9.9.9.9:53",
		"-socks-proxy", "proxy",
		"-socks-proxy-port", "1081",
		"-video-w", "640",
		"-video-h", "480",
		"-video-fps", "30",
		"-video-bitrate", "1M",
		"-video-hw", "none",
		"-video-qr-size", "128",
		"-video-qr-recovery", "high",
		"-video-codec", "tile",
		"-video-tile-module", "6",
		"-video-tile-rs", "40",
		"-vp8-fps", "24",
		"-vp8-batch", "3",
		"-fps", "40",
		"-batch", "4",
		"-frag", "512",
		"-ack-ms", "1500",
		"-lifetime", "300",
	}, flag.ContinueOnError)
	if err != nil {
		t.Fatalf("parseFlagsFrom() error = %v", err)
	}
	if cfg.mode != "srv" || cfg.carrier != "telemost" || cfg.roomID != "room" ||
		cfg.debug != true || cfg.videoCodec != "tile" || cfg.videoTileRS != 40 ||
		cfg.vp8FPS != 24 || cfg.vp8BatchSize != 3 || cfg.seiFPS != 40 ||
		cfg.seiBatchSize != 4 || cfg.seiFragmentSize != 512 || cfg.seiAckTimeoutMS != 1500 ||
		cfg.lifetime != 300 {
		t.Fatalf("parseFlagsFrom() = %+v", cfg)
	}

	_, err = parseFlagsFrom([]string{"-bad"}, flag.ContinueOnError)
	if err == nil {
		t.Fatal("parseFlagsFrom(bad flag) error = nil")
	}
}

func TestRunWithConfigValidationAndDataDirErrors(t *testing.T) {
	session.RegisterDefaults()
	cfg := config{
		mode:       "srv",
		link:       "direct",
		transport:  "datachannel",
		carrier:    "jazz",
		clientID:   "client",
		keyHex:     "key",
		dnsServer:  "1.1.1.1:53",
		videoCodec: "qrcode",
	}
	if err := runWithConfig(cfg); !errors.Is(err, ErrDataDirRequired) {
		t.Fatalf("runWithConfig(no data dir) = %v, want %v", err, ErrDataDirRequired)
	}

	cfg.mode = ""
	if err := runWithConfig(cfg); err == nil {
		t.Fatal("runWithConfig(invalid config) error = nil")
	}
}

func TestRunWithArgsSuccessfulSessionReturn(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "names"), []byte("A\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(names) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "surnames"), []byte("B\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(surnames) error = %v", err)
	}

	oldRunSession := runSession
	t.Cleanup(func() {
		runSession = oldRunSession
	})
	called := false
	runSession = func(ctx context.Context, cfg session.Config) error {
		called = true
		if cfg.Mode != "srv" || cfg.Carrier != "jazz" || cfg.ClientID != "client" {
			t.Fatalf("session config = %+v", cfg)
		}
		select {
		case <-ctx.Done():
			t.Fatal("context canceled before session returned")
		default:
		}
		return nil
	}

	err := runWithArgs([]string{
		"-mode", "srv",
		"-link", "direct",
		"-transport", "datachannel",
		"-carrier", "jazz",
		"-client-id", "client",
		"-key", "key",
		"-dns", "1.1.1.1:53",
		"-data", dir,
	})
	if err != nil {
		t.Fatalf("runWithArgs() error = %v", err)
	}
	if !called {
		t.Fatal("runWithArgs() did not call session runner")
	}
}

func TestConfigureLogging(t *testing.T) {
	t.Setenv("PION_LOG_DISABLE", "")
	logger.SetVerbose(false)
	configureLogging(true)
	if !logger.IsVerbose() {
		t.Fatal("configureLogging(true) did not enable verbose logging")
	}
	if got := os.Getenv("PION_LOG_DISABLE"); got != "" {
		t.Fatalf("configureLogging(true) PION_LOG_DISABLE = %q, want empty", got)
	}

	logger.SetVerbose(false)
	configureLogging(false)
	if logger.IsVerbose() {
		t.Fatal("configureLogging(false) enabled verbose logging")
	}
	if got := os.Getenv("PION_LOG_DISABLE"); got != "all" {
		t.Fatalf("configureLogging(false) PION_LOG_DISABLE = %q, want all", got)
	}
}

func TestResolveDataDir(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "data")
	got, err := resolveDataDir(abs)
	if err != nil {
		t.Fatalf("resolveDataDir(abs) error = %v", err)
	}
	if got != abs {
		t.Fatalf("resolveDataDir(abs) = %q, want %q", got, abs)
	}

	got, err = resolveDataDir("data")
	if err != nil {
		t.Fatalf("resolveDataDir(rel) error = %v", err)
	}
	if filepath.Base(got) != "data" || !filepath.IsAbs(got) {
		t.Fatalf("resolveDataDir(rel) = %q, want absolute path ending in data", got)
	}
}

func TestLoadNames(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "names"), []byte("A\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(names) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "surnames"), []byte("B\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(surnames) error = %v", err)
	}
	if err := loadNames(dir); err != nil {
		t.Fatalf("loadNames() error = %v", err)
	}
}

func TestWaitForShutdown(t *testing.T) {
	errCh := make(chan error, 1)
	errCh <- nil
	if err := waitForShutdown(errCh, 1); err != nil {
		t.Fatalf("waitForShutdown(nil) error = %v", err)
	}

	want := errors.New("boom")
	errCh = make(chan error, 1)
	errCh <- want
	if err := waitForShutdown(errCh, 1); !errors.Is(err, want) {
		t.Fatalf("waitForShutdown(error) = %v, want %v", err, want)
	}
}

func TestLoadJSONConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "olcrtc.json")
	data := []byte(`{
		"mode": "cnc",
		"link": "direct",
		"endpoint": {
			"room_id": "room-id",
			"key": "64_hex_key"
		},
		"client-id": "client",
		"carrier": "wbstream",
		"transport": {
			"type": "seichannel",
			"vp8": {
				"fps": 60,
				"batch": 64
			},
			"sei": {
				"fps": 40,
				"batch": 4,
				"frag": 512,
				"ack_ms": 1500
			}
		},
		"dns": "1.1.1.1:53",
		"data": "data",
		"client": {
			"socks_host": "127.0.0.1",
			"socks_port": 1080
		},
		"lifetime": 300,
		"port": 8080
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadJSONConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.mode != "cnc" || cfg.link != "direct" || cfg.roomID != "room-id" ||
		cfg.clientID != "client" || cfg.keyHex != "64_hex_key" || cfg.carrier != "wbstream" ||
		cfg.transport != "seichannel" || cfg.vp8FPS != 60 || cfg.vp8BatchSize != 64 ||
		cfg.seiFPS != 40 || cfg.seiBatchSize != 4 || cfg.seiFragmentSize != 512 ||
		cfg.seiAckTimeoutMS != 1500 || cfg.dnsServer != "1.1.1.1:53" ||
		cfg.dataDir != "data" || cfg.socksHost != "127.0.0.1" || cfg.socksPort != 1080 ||
		cfg.lifetime != 300 || cfg.videoQRRecovery != "low" || cfg.videoCodec != "qrcode" {
		t.Fatalf("loadJSONConfig() = %+v", cfg)
	}

	loaded, err := loadJSONConfigFile(path)
	if err != nil {
		t.Fatalf("load config file: %v", err)
	}
	if loaded.port != 8080 {
		t.Fatalf("port = %d, want 8080", loaded.port)
	}
}

func TestLoadJSONConfigsArray(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "server.json")
	data := []byte(`[
		{
			"label": "vp8",
			"endpoint": {
				"room_id": "any",
				"key": "e830d36f7be8cfb04a741fc1a5e2ddf8ff04f30985dc070616483f939ad5fafe"
			},
			"carrier": "wbstream",
			"transport": {
				"type": "vp8channel",
				"vp8": {
					"fps": 60,
					"batch": 64
				}
			}
		},
		{
			"label": "data",
			"endpoint": {
				"room_id": "room-2",
				"key": "e830d36f7be8cfb04a741fc1a5e2ddf8ff04f30985dc070616483f939ad5fafe"
			},
			"carrier": "telemost",
			"transport": {
				"type": "vp8channel",
				"vp8": {
					"fps": 30,
					"batch": 8
				}
			}
		}
	]`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfgs, err := loadJSONConfigs(path)
	if err != nil {
		t.Fatalf("load configs: %v", err)
	}

	if len(cfgs) != 2 {
		t.Fatalf("len(cfgs) = %d, want 2", len(cfgs))
	}
	if cfgs[0].roomID != "any" {
		t.Fatalf("cfgs[0].roomID = %q, want any", cfgs[0].roomID)
	}
	if cfgs[0].label != "vp8" {
		t.Fatalf("cfgs[0].label = %q, want vp8", cfgs[0].label)
	}
	if cfgs[0].carrier != "wbstream" {
		t.Fatalf("cfgs[0].carrier = %q, want wbstream", cfgs[0].carrier)
	}
	if cfgs[0].transport != "vp8channel" {
		t.Fatalf("cfgs[0].transport = %q, want vp8channel", cfgs[0].transport)
	}
	if cfgs[0].vp8FPS != 60 {
		t.Fatalf("cfgs[0].vp8FPS = %d, want 60", cfgs[0].vp8FPS)
	}
	if cfgs[0].vp8BatchSize != 64 {
		t.Fatalf("cfgs[0].vp8BatchSize = %d, want 64", cfgs[0].vp8BatchSize)
	}
	if cfgs[1].roomID != "room-2" {
		t.Fatalf("cfgs[1].roomID = %q, want room-2", cfgs[1].roomID)
	}
	if cfgs[1].label != "data" {
		t.Fatalf("cfgs[1].label = %q, want data", cfgs[1].label)
	}
	if cfgs[1].carrier != "telemost" {
		t.Fatalf("cfgs[1].carrier = %q, want telemost", cfgs[1].carrier)
	}
	if cfgs[1].vp8FPS != 30 {
		t.Fatalf("cfgs[1].vp8FPS = %d, want 30", cfgs[1].vp8FPS)
	}
	if cfgs[1].vp8BatchSize != 8 {
		t.Fatalf("cfgs[1].vp8BatchSize = %d, want 8", cfgs[1].vp8BatchSize)
	}
}

func TestLoadJSONConfigsVersion4(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "client.json")
	data := []byte(`{
		"version": 4,
		"active_location_id": "netherlands",
		"port": 9090,
		"name": "Main subscription",
		"update": 1778011200,
		"refresh": "10m",
		"color": "#4A90E2",
		"icon": "flag",
		"used": "10mb/10gb",
		"available": "9.99gb",
		"locations": [
			{
				"storage_id": "germany",
				"name": "Germany",
				"color": "#111111",
				"icon": "de",
				"used": "1mb/10gb",
				"available": "9gb",
				"ip": "203.0.113.10",
				"comment": "primary node",
				"mimo": "DE / primary / IPv4",
				"endpoint": {
					"room_id": "room-1",
					"key": "key-1"
				},
				"carrier": "wbstream",
				"transport": {
					"type": "datachannel"
				}
			},
			{
				"storage_id": "netherlands",
				"name": "Netherlands",
				"endpoint": {
					"room_id": "room-2",
					"key": "key-2"
				},
				"carrier": "telemost",
				"transport": {
					"type": "vp8channel"
				}
			}
		]
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := loadJSONConfigFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.activeLocationID != "netherlands" {
		t.Fatalf("activeLocationID = %q, want netherlands", loaded.activeLocationID)
	}
	if loaded.port != 9090 {
		t.Fatalf("port = %d, want 9090", loaded.port)
	}
	if loaded.subscription.Name != "Main subscription" || loaded.subscription.Update != 1778011200 ||
		loaded.subscription.Refresh != "10m" || loaded.subscription.Color != "#4A90E2" ||
		loaded.subscription.Icon != "flag" || loaded.subscription.Used != "10mb/10gb" ||
		loaded.subscription.Available != "9.99gb" {
		t.Fatalf("subscription = %+v", loaded.subscription)
	}
	if len(loaded.locations) != 2 {
		t.Fatalf("len(locations) = %d, want 2", len(loaded.locations))
	}
	if loaded.locations[1].storageID != "netherlands" {
		t.Fatalf("storageID = %q, want netherlands", loaded.locations[1].storageID)
	}
	if loaded.locations[1].label != "Netherlands" {
		t.Fatalf("label = %q, want Netherlands", loaded.locations[1].label)
	}
	if loaded.locations[1].roomID != "room-2" {
		t.Fatalf("roomID = %q, want room-2", loaded.locations[1].roomID)
	}
	if loaded.locations[0].color != "#111111" || loaded.locations[0].icon != "de" ||
		loaded.locations[0].used != "1mb/10gb" || loaded.locations[0].available != "9gb" ||
		loaded.locations[0].ip != "203.0.113.10" || loaded.locations[0].comment != "primary node" ||
		loaded.locations[0].mimo != "DE / primary / IPv4" {
		t.Fatalf("location metadata = %+v", loaded.locations[0])
	}
}

func TestClientConfigHandlerServesCurrentSubscription(t *testing.T) {
	oldUnixNow := unixNow
	unixNow = func() int64 { return 1778011200 }
	t.Cleanup(func() { unixNow = oldUnixNow })

	dataCfg := defaultConfig()
	dataCfg.storageID = "amsterdam-wb-dc"
	dataCfg.label = "Netherlands"
	dataCfg.roomID = "old-room"
	dataCfg.clientID = "user"
	dataCfg.keyHex = "key"
	dataCfg.carrier = "wbstream"
	dataCfg.transport = "datachannel"
	dataCfg.color = "#4A90E2"
	dataCfg.icon = "nl"
	dataCfg.used = "500mb/10gb"
	dataCfg.available = "9.5gb"
	dataCfg.ip = "203.0.113.10"
	dataCfg.comment = "basic free node"
	dataCfg.mimo = "NL / olcng free sub / IPv6"
	dataCfg.lifetime = 600

	store := newServedConfigStore([]config{dataCfg}, subscriptionMetadata{
		Name:      "Zarazaex Free RU",
		Refresh:   "10m",
		Color:     "#4A90E2",
		Icon:      "sub",
		Used:      "10mb/10gb",
		Available: "9.99gb",
	})
	store.setRoomID(0)("new-room")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	clientConfigHandler(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("content type = %q, want text/plain; charset=utf-8", got)
	}

	body := rec.Body.String()
	if strings.Contains(body, `"port"`) || strings.Contains(body, `"version"`) {
		t.Fatalf("response is not subscription text: %s", body)
	}
	wantParts := []string{
		"#name: Zarazaex Free RU",
		"#update: 1778011200",
		"#refresh: 1778011800",
		"#color: #4A90E2",
		"#icon: sub",
		"#used: 10mb/10gb",
		"#available: 9.99gb",
		"olcrtc://wbstream?datachannel@new-room#key%user$NL / olcng free sub / IPv6",
		"##name: Netherlands",
		"##color: #4A90E2",
		"##icon: nl",
		"##used: 500mb/10gb",
		"##available: 9.5gb",
		"##ip: 203.0.113.10",
		"##comment: basic free node",
	}
	for _, want := range wantParts {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %q:\n%s", want, body)
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	clientConfigHandler(store).ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("post status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestSubscriptionTimestampsWithLocationLifetime(t *testing.T) {
	oldUnixNow := unixNow
	unixNow = func() int64 { return 1778012000 }
	t.Cleanup(func() { unixNow = oldUnixNow })

	store := newServedConfigStore([]config{
		{
			label:     "Netherlands",
			roomID:    "room-1",
			clientID:  "user",
			keyHex:    "key",
			carrier:   "wbstream",
			transport: "datachannel",
			lifetime:  600,
		},
		{
			label:     "Netherlands",
			roomID:    "room-2",
			clientID:  "user",
			keyHex:    "key",
			carrier:   "wbstream",
			transport: "vp8channel",
		},
	}, subscriptionMetadata{Name: "ScumVPN"})

	body := store.subscriptionText()
	for _, want := range []string{
		"#name: ScumVPN",
		"#update: 1778012000",
		"#refresh: 1778012600",
		"olcrtc://wbstream?datachannel@room-1#key%user$",
		"olcrtc://wbstream?vp8channel@room-2#key%user$",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("subscription missing %q:\n%s", want, body)
		}
	}
}

func TestSelectActiveLocationOnlyForClient(t *testing.T) {
	t.Parallel()

	cfgs := []config{
		{storageID: "germany", mode: "cnc", roomID: "room-1"},
		{storageID: "netherlands", mode: "cnc", roomID: "room-2"},
	}

	selected, err := selectActiveLocation(cfgs, "netherlands")
	if err != nil {
		t.Fatalf("select active location: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("len(selected) = %d, want 1", len(selected))
	}
	if selected[0].roomID != "room-2" {
		t.Fatalf("selected roomID = %q, want room-2", selected[0].roomID)
	}

	cfgs[0].mode = "srv"
	cfgs[1].mode = "srv"
	selected, err = selectActiveLocation(cfgs, "netherlands")
	if err != nil {
		t.Fatalf("select server locations: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("len(server selected) = %d, want 2", len(selected))
	}
}

func TestMergeConfigAppliesOnlyExplicitFlags(t *testing.T) {
	dst := config{
		carrier:      "wbstream",
		transport:    "vp8channel",
		vp8FPS:       60,
		vp8BatchSize: 64,
		seiFPS:       40,
	}
	flags := config{
		carrier:      "telemost",
		transport:    "",
		vp8FPS:       30,
		vp8BatchSize: 0,
		seiFPS:       20,
	}

	mergeConfig(&dst, flags, map[string]bool{
		"carrier": true,
		"vp8-fps": true,
		"fps":     true,
	})

	if dst.carrier != "telemost" || dst.transport != "vp8channel" ||
		dst.vp8FPS != 30 || dst.vp8BatchSize != 64 || dst.seiFPS != 20 {
		t.Fatalf("mergeConfig() = %+v", dst)
	}
}

func TestParseFlagsFromJSONConfigWithOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "olcrtc.json")
	data := []byte(`{
		"mode": "cnc",
		"link": "direct",
		"id": "room-id",
		"client_id": "client",
		"key": "key",
		"carrier": "wbstream",
		"transport": {
			"type": "vp8channel",
			"vp8": {"fps": 60, "batch": 64}
		},
		"dns": "1.1.1.1:53",
		"data": "data"
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := parseFlagsFrom([]string{
		"-config", path,
		"-carrier", "telemost",
		"-vp8-fps", "30",
		"-client-id", "override-client",
	}, flag.ContinueOnError)
	if err != nil {
		t.Fatalf("parseFlagsFrom(config) error = %v", err)
	}

	if cfg.carrier != "telemost" || cfg.vp8FPS != 30 ||
		cfg.vp8BatchSize != 64 || cfg.clientID != "override-client" {
		t.Fatalf("parseFlagsFrom(config) = %+v", cfg)
	}
}

func TestRuntimeConfigDataDirRequiresSameValue(t *testing.T) {
	t.Parallel()

	cfg := runtimeConfig{locations: []config{
		{dataDir: "data"},
		{dataDir: "data"},
	}}
	dataDir, err := cfg.dataDir()
	if err != nil {
		t.Fatalf("dataDir: %v", err)
	}
	if dataDir != "data" {
		t.Fatalf("dataDir = %q, want data", dataDir)
	}

	cfg.locations[1].dataDir = "other"
	if _, err := cfg.dataDir(); err == nil {
		t.Fatal("dataDir with mismatched locations succeeded")
	}
}
