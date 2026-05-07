// Package main provides the olcrtc CLI entrypoint.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	protoLogger "github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/openlibrecommunity/olcrtc/internal/app/session"
	"github.com/openlibrecommunity/olcrtc/internal/logger"
	"github.com/openlibrecommunity/olcrtc/internal/names"
)

// ErrDataDirRequired is returned when no data directory is specified.
var ErrDataDirRequired = errors.New("data directory required (use -data data)")

//nolint:gochecknoglobals // Tests replace the long-running session runner with a bounded function.
var runSession = session.Run

type config struct {
	label           string
	mode            string
	link            string
	transport       string
	carrier         string
	roomID          string
	clientID        string
	provider        string
	socksPort       int
	socksHost       string
	keyHex          string
	debug           bool
	dataDir         string
	dnsServer       string
	socksProxyAddr  string
	socksProxyPort  int
	videoWidth      int
	videoHeight     int
	videoFPS        int
	videoBitrate    string
	videoHW         string
	videoQRSize     int
	videoQRRecovery string
	videoCodec      string
	videoTileModule int
	videoTileRS     int
	vp8FPS          int
	vp8BatchSize    int
	seiFPS          int
	seiBatchSize    int
	seiFragmentSize int
	seiAckTimeoutMS int
	lifetime        int
}

type runtimeConfig struct {
	locations []config
}

func main() {
	if err := run(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func run() error {
	return runWithArgs(os.Args[1:])
}

func runWithArgs(args []string) error {
	session.RegisterDefaults()

	runtimeCfg, err := parseRuntimeFlagsFrom(args, flag.ExitOnError)
	if err != nil {
		return err
	}
	configureLogging(runtimeCfg.debug())

	for i, cfg := range runtimeCfg.locations {
		if err := session.Validate(toSessionConfig(cfg)); err != nil {
			return fmt.Errorf("validate config location %d: %w", i+1, err)
		}
	}

	dataDir, err := runtimeCfg.dataDir()
	if err != nil {
		return err
	}

	resolvedDataDir, err := resolveDataDir(dataDir)
	if err != nil {
		return err
	}

	if err := loadNames(resolvedDataDir); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, len(runtimeCfg.locations))
	for i, cfg := range runtimeCfg.locations {
		i, cfg := i, cfg
		go func() {
			logger.Infof("Starting location %d/%d [%s]: %s/%s/%s room=%s",
				i+1, len(runtimeCfg.locations), cfg.label, cfg.link, cfg.transport, cfg.carrier, cfg.roomID)
			if err := runSession(ctx, toSessionConfig(cfg)); err != nil {
				errCh <- fmt.Errorf("location %d: %w", i+1, err)
				return
			}
			errCh <- nil
		}()
	}

	select {
	case <-sigCh:
		logger.Info("Shutting down gracefully...")
		cancel()
		return waitForShutdown(errCh, len(runtimeCfg.locations))
	case err := <-errCh:
		cancel()
		return err
	}
}

func runWithConfig(cfg config) error {
	return runWithRuntimeConfig(runtimeConfig{locations: []config{cfg}})
}

func runWithRuntimeConfig(runtimeCfg runtimeConfig) error {
	configureLogging(runtimeCfg.debug())

	for i, cfg := range runtimeCfg.locations {
		if err := session.Validate(toSessionConfig(cfg)); err != nil {
			return fmt.Errorf("validate config location %d: %w", i+1, err)
		}
	}

	dataDir, err := runtimeCfg.dataDir()
	if err != nil {
		return err
	}

	resolvedDataDir, err := resolveDataDir(dataDir)
	if err != nil {
		return err
	}

	if err := loadNames(resolvedDataDir); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, len(runtimeCfg.locations))
	for i, cfg := range runtimeCfg.locations {
		i, cfg := i, cfg
		go func() {
			if err := runSession(ctx, toSessionConfig(cfg)); err != nil {
				errCh <- fmt.Errorf("location %d: %w", i+1, err)
				return
			}
			errCh <- nil
		}()
	}

	return waitForShutdown(errCh, len(runtimeCfg.locations))
}

func parseFlags() config {
	cfg, _ := parseFlagsFrom(os.Args[1:], flag.ExitOnError)
	return cfg
}

func (c runtimeConfig) debug() bool {
	for _, cfg := range c.locations {
		if cfg.debug {
			return true
		}
	}
	return false
}

func (c runtimeConfig) dataDir() (string, error) {
	dataDir := ""
	for i, cfg := range c.locations {
		if cfg.dataDir == "" {
			return "", fmt.Errorf("location %d: %w", i+1, ErrDataDirRequired)
		}
		if dataDir == "" {
			dataDir = cfg.dataDir
			continue
		}
		if cfg.dataDir != dataDir {
			return "", fmt.Errorf("location %d: data directory %q differs from %q",
				i+1, cfg.dataDir, dataDir)
		}
	}
	if dataDir == "" {
		return "", ErrDataDirRequired
	}
	return dataDir, nil
}

func defaultConfig() config {
	return config{
		videoQRRecovery: "low",
		videoCodec:      "qrcode",
	}
}

func applyLocationLabels(cfgs []config) {
	for i := range cfgs {
		if cfgs[i].label == "" {
			cfgs[i].label = fmt.Sprintf("location %d", i+1)
		}
	}
}

type jsonConfig struct {
	Label     string        `json:"label"`
	Mode      string        `json:"mode"`
	Link      string        `json:"link"`
	Carrier   string        `json:"carrier"`
	Provider  string        `json:"provider"`
	Endpoint  jsonEndpoint  `json:"endpoint"`
	Transport jsonTransport `json:"transport"`
	DNS       string        `json:"dns"`
	Data      string        `json:"data"`
	Debug     bool          `json:"debug"`
	Client    jsonClient    `json:"client"`
	Server    jsonServer    `json:"server"`
	Video     jsonVideo     `json:"video"`
	SEI       jsonSEI       `json:"sei"`
	Lifetime  int           `json:"lifetime"`

	RoomID         string `json:"id"`
	ClientID       string `json:"client_id"`
	Key            string `json:"key"`
	SOCKSHost      string `json:"socks_host"`
	SOCKSPort      int    `json:"socks_port"`
	SOCKSProxy     string `json:"socks_proxy"`
	SOCKSProxyPort int    `json:"socks_proxy_port"`
	SEIFPS         int    `json:"fps"`
	SEIBatch       int    `json:"batch"`
	SEIFragment    int    `json:"frag"`
	SEIAckMS       int    `json:"ack_ms"`
}

type jsonEndpoint struct {
	RoomID string `json:"room_id"`
	Key    string `json:"key"`
}

type jsonTransport struct {
	Type  string    `json:"type"`
	VP8   jsonVP8   `json:"vp8"`
	Video jsonVideo `json:"video"`
	SEI   jsonSEI   `json:"sei"`
}

type jsonVP8 struct {
	FPS   int `json:"fps"`
	Batch int `json:"batch"`
}

type jsonVideo struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	FPS        int    `json:"fps"`
	Bitrate    string `json:"bitrate"`
	HW         string `json:"hw"`
	QRSize     int    `json:"qr_size"`
	QRRecovery string `json:"qr_recovery"`
	Codec      string `json:"codec"`
	TileModule int    `json:"tile_module"`
	TileRS     int    `json:"tile_rs"`
}

type jsonSEI struct {
	FPS      int `json:"fps"`
	Batch    int `json:"batch"`
	Fragment int `json:"frag"`
	AckMS    int `json:"ack_ms"`
}

type jsonClient struct {
	SOCKSHost string `json:"socks_host"`
	SOCKSPort int    `json:"socks_port"`
}

type jsonServer struct {
	SOCKSProxy     string `json:"socks_proxy"`
	SOCKSProxyPort int    `json:"socks_proxy_port"`
}

func loadJSONConfigs(path string) ([]config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var body json.RawMessage
	if err := dec.Decode(&body); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("decode config: multiple JSON values")
		}
		return nil, fmt.Errorf("decode config: %w", err)
	}

	var raws []jsonConfig
	switch {
	case len(body) == 0:
		return nil, errors.New("decode config: empty config")
	case bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")):
		if err := decodeStrict(body, &raws); err != nil {
			return nil, fmt.Errorf("decode config: %w", err)
		}
		if len(raws) == 0 {
			return nil, errors.New("decode config: empty location list")
		}
	default:
		var raw jsonConfig
		if err := decodeStrict(body, &raw); err != nil {
			return nil, fmt.Errorf("decode config: %w", err)
		}
		raws = []jsonConfig{raw}
	}

	cfgs := make([]config, 0, len(raws))
	for _, raw := range raws {
		cfg := defaultConfig()
		applyJSONConfig(&cfg, raw)
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}

func loadJSONConfig(path string) (config, error) {
	cfgs, err := loadJSONConfigs(path)
	if err != nil {
		return config{}, err
	}
	return cfgs[0], nil
}

func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func applyJSONConfig(cfg *config, raw jsonConfig) {
	setString(&cfg.label, raw.Label)
	setString(&cfg.mode, raw.Mode)
	setString(&cfg.link, raw.Link)
	setString(&cfg.carrier, raw.Carrier)
	setString(&cfg.provider, raw.Provider)
	setString(&cfg.roomID, firstNonEmpty(raw.Endpoint.RoomID, raw.RoomID))
	setString(&cfg.clientID, raw.ClientID)
	setString(&cfg.keyHex, firstNonEmpty(raw.Endpoint.Key, raw.Key))
	setString(&cfg.transport, raw.Transport.Type)
	setString(&cfg.dnsServer, raw.DNS)
	setString(&cfg.dataDir, raw.Data)
	if raw.Debug {
		cfg.debug = true
	}
	setString(&cfg.socksHost, firstNonEmpty(raw.Client.SOCKSHost, raw.SOCKSHost))
	setInt(&cfg.socksPort, firstNonZero(raw.Client.SOCKSPort, raw.SOCKSPort))
	setString(&cfg.socksProxyAddr, firstNonEmpty(raw.Server.SOCKSProxy, raw.SOCKSProxy))
	setInt(&cfg.socksProxyPort, firstNonZero(raw.Server.SOCKSProxyPort, raw.SOCKSProxyPort))

	video := raw.Video
	if raw.Transport.Video != (jsonVideo{}) {
		video = raw.Transport.Video
	}
	setInt(&cfg.videoWidth, video.Width)
	setInt(&cfg.videoHeight, video.Height)
	setInt(&cfg.videoFPS, video.FPS)
	setString(&cfg.videoBitrate, video.Bitrate)
	setString(&cfg.videoHW, video.HW)
	setInt(&cfg.videoQRSize, video.QRSize)
	setString(&cfg.videoQRRecovery, video.QRRecovery)
	setString(&cfg.videoCodec, video.Codec)
	setInt(&cfg.videoTileModule, video.TileModule)
	setInt(&cfg.videoTileRS, video.TileRS)

	setInt(&cfg.vp8FPS, raw.Transport.VP8.FPS)
	setInt(&cfg.vp8BatchSize, raw.Transport.VP8.Batch)

	sei := raw.SEI
	if raw.Transport.SEI != (jsonSEI{}) {
		sei = raw.Transport.SEI
	}
	setInt(&cfg.seiFPS, firstNonZero(sei.FPS, raw.SEIFPS))
	setInt(&cfg.seiBatchSize, firstNonZero(sei.Batch, raw.SEIBatch))
	setInt(&cfg.seiFragmentSize, firstNonZero(sei.Fragment, raw.SEIFragment))
	setInt(&cfg.seiAckTimeoutMS, firstNonZero(sei.AckMS, raw.SEIAckMS))
	setInt(&cfg.lifetime, raw.Lifetime)
}

func mergeConfig(dst *config, flags config, setFlags map[string]bool) {
	if setFlags["label"] {
		dst.label = flags.label
	}
	if setFlags["mode"] {
		dst.mode = flags.mode
	}
	if setFlags["link"] {
		dst.link = flags.link
	}
	if setFlags["transport"] {
		dst.transport = flags.transport
	}
	if setFlags["carrier"] {
		dst.carrier = flags.carrier
	}
	if setFlags["id"] {
		dst.roomID = flags.roomID
	}
	if setFlags["client-id"] {
		dst.clientID = flags.clientID
	}
	if setFlags["provider"] {
		dst.provider = flags.provider
	}
	if setFlags["socks-port"] {
		dst.socksPort = flags.socksPort
	}
	if setFlags["socks-host"] {
		dst.socksHost = flags.socksHost
	}
	if setFlags["key"] {
		dst.keyHex = flags.keyHex
	}
	if setFlags["debug"] {
		dst.debug = flags.debug
	}
	if setFlags["data"] {
		dst.dataDir = flags.dataDir
	}
	if setFlags["dns"] {
		dst.dnsServer = flags.dnsServer
	}
	if setFlags["socks-proxy"] {
		dst.socksProxyAddr = flags.socksProxyAddr
	}
	if setFlags["socks-proxy-port"] {
		dst.socksProxyPort = flags.socksProxyPort
	}
	if setFlags["video-w"] {
		dst.videoWidth = flags.videoWidth
	}
	if setFlags["video-h"] {
		dst.videoHeight = flags.videoHeight
	}
	if setFlags["video-fps"] {
		dst.videoFPS = flags.videoFPS
	}
	if setFlags["video-bitrate"] {
		dst.videoBitrate = flags.videoBitrate
	}
	if setFlags["video-hw"] {
		dst.videoHW = flags.videoHW
	}
	if setFlags["video-qr-size"] {
		dst.videoQRSize = flags.videoQRSize
	}
	if setFlags["video-qr-recovery"] {
		dst.videoQRRecovery = flags.videoQRRecovery
	}
	if setFlags["video-codec"] {
		dst.videoCodec = flags.videoCodec
	}
	if setFlags["video-tile-module"] {
		dst.videoTileModule = flags.videoTileModule
	}
	if setFlags["video-tile-rs"] {
		dst.videoTileRS = flags.videoTileRS
	}
	if setFlags["vp8-fps"] {
		dst.vp8FPS = flags.vp8FPS
	}
	if setFlags["vp8-batch"] {
		dst.vp8BatchSize = flags.vp8BatchSize
	}
	if setFlags["fps"] {
		dst.seiFPS = flags.seiFPS
	}
	if setFlags["batch"] {
		dst.seiBatchSize = flags.seiBatchSize
	}
	if setFlags["frag"] {
		dst.seiFragmentSize = flags.seiFragmentSize
	}
	if setFlags["ack-ms"] {
		dst.seiAckTimeoutMS = flags.seiAckTimeoutMS
	}
	if setFlags["lifetime"] {
		dst.lifetime = flags.lifetime
	}
}

func setString(dst *string, value string) {
	if value != "" {
		*dst = value
	}
}

func setInt(dst *int, value int) {
	if value != 0 {
		*dst = value
	}
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseFlagsFrom(args []string, errorHandling flag.ErrorHandling) (config, error) {
	runtimeCfg, err := parseRuntimeFlagsFrom(args, errorHandling)
	if err != nil {
		return config{}, err
	}
	return runtimeCfg.locations[0], nil
}

func parseRuntimeFlagsFrom(args []string, errorHandling flag.ErrorHandling) (runtimeConfig, error) {
	cfg := defaultConfig()
	configFile := ""
	fs := flag.NewFlagSet("olcrtc", errorHandling)
	if errorHandling == flag.ContinueOnError {
		fs.SetOutput(io.Discard)
	}

	fs.StringVar(&configFile, "config", "", "Path to JSON config file")
	fs.StringVar(&cfg.label, "label", "", "Location label used in logs")
	fs.StringVar(&cfg.mode, "mode", "", "Mode: srv or cnc")
	fs.StringVar(&cfg.link, "link", "", "Link: direct (p2p connection type)")
	fs.StringVar(&cfg.transport, "transport", "", "Transport: datachannel, videochannel, seichannel, vp8channel")
	fs.StringVar(&cfg.carrier, "carrier", "", "Carrier: telemost, jazz, wbstream")
	fs.StringVar(&cfg.roomID, "id", "", "Room ID")
	fs.StringVar(&cfg.clientID, "client-id", "", "Client ID: binds one srv to one cnc (required)")
	fs.StringVar(&cfg.provider, "provider", "", "Deprecated alias for -carrier")
	fs.IntVar(&cfg.socksPort, "socks-port", 0, "SOCKS5 port (client only)")
	fs.StringVar(&cfg.socksHost, "socks-host", "", "SOCKS5 listen host (client only)")
	fs.StringVar(&cfg.keyHex, "key", "", "Shared encryption key (hex)")
	fs.BoolVar(&cfg.debug, "debug", false, "Enable verbose logging")
	fs.StringVar(&cfg.dataDir, "data", "", "Path to data directory")
	fs.StringVar(&cfg.dnsServer, "dns", "", "DNS server (e.g. 1.1.1.1:53)")
	fs.StringVar(&cfg.socksProxyAddr, "socks-proxy", "", "SOCKS5 proxy address (server only)")
	fs.IntVar(&cfg.socksProxyPort, "socks-proxy-port", 0, "SOCKS5 proxy port (server only)")
	fs.IntVar(&cfg.videoWidth, "video-w", 0, "Video logical width (videochannel only)")
	fs.IntVar(&cfg.videoHeight, "video-h", 0, "Video logical height (videochannel only)")
	fs.IntVar(&cfg.videoFPS, "video-fps", 0, "Video frames per second (videochannel only)")
	fs.StringVar(&cfg.videoBitrate, "video-bitrate", "", "Video bitrate (videochannel only)")
	fs.StringVar(&cfg.videoHW, "video-hw", "", "Hardware acceleration (none, nvenc)")
	fs.IntVar(&cfg.videoQRSize, "video-qr-size", 0, "Video QR code fragment size (videochannel only)")
	fs.StringVar(&cfg.videoQRRecovery, "video-qr-recovery", "low",
		"QR error correction: low (7%), medium (15%), high (25%), highest (30%)")
	fs.StringVar(&cfg.videoCodec, "video-codec", "qrcode", "Visual codec: qrcode or tile")
	fs.IntVar(&cfg.videoTileModule, "video-tile-module", 0,
		"Tile module size in pixels 1..270 (videochannel tile only, default 4)")
	fs.IntVar(&cfg.videoTileRS, "video-tile-rs", 0,
		"Tile Reed-Solomon parity percent 0..200 (videochannel tile only, default 20)")
	fs.IntVar(&cfg.vp8FPS, "vp8-fps", 0, "VP8 frames per second (vp8channel only, default 25)")
	fs.IntVar(&cfg.vp8BatchSize, "vp8-batch", 0, "VP8 frames per tick (vp8channel only, default 1)")
	fs.IntVar(&cfg.seiFPS, "fps", 0, "Frames per second for transports that use video timing (seichannel)")
	fs.IntVar(&cfg.seiBatchSize, "batch", 0, "Transport frames per tick for batched transports (seichannel)")
	fs.IntVar(&cfg.seiFragmentSize, "frag", 0, "Fragment size in bytes for fragmented transports (seichannel)")
	fs.IntVar(&cfg.seiAckTimeoutMS, "ack-ms", 0, "ACK timeout in milliseconds for reliable visual transports (seichannel)")
	fs.IntVar(&cfg.lifetime, "lifetime", 0, "Room lifetime in seconds (server only, 0 = infinite)")

	if err := fs.Parse(args); err != nil {
		return runtimeConfig{}, err
	}
	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})
	if configFile != "" {
		fileCfgs, err := loadJSONConfigs(configFile)
		if err != nil {
			return runtimeConfig{}, err
		}
		for i := range fileCfgs {
			mergeConfig(&fileCfgs[i], cfg, setFlags)
			normalizeConfig(&fileCfgs[i])
		}
		applyLocationLabels(fileCfgs)
		return runtimeConfig{locations: fileCfgs}, nil
	}
	normalizeConfig(&cfg)
	cfgs := []config{cfg}
	applyLocationLabels(cfgs)
	return runtimeConfig{locations: cfgs}, nil
}

func normalizeConfig(cfg *config) {
	if cfg.carrier == "" {
		cfg.carrier = cfg.provider
	}
}

func configureLogging(debug bool) {
	if debug {
		logger.SetVerbose(true)
		return
	}
	// Suppress noisy LiveKit/pion logs unless debug is enabled.
	_ = os.Setenv("PION_LOG_DISABLE", "all")
	lksdk.SetLogger(protoLogger.GetDiscardLogger())
}

func resolveDataDir(dataDir string) (string, error) {
	if filepath.IsAbs(dataDir) {
		return dataDir, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	return filepath.Join(filepath.Dir(exePath), dataDir), nil
}

func loadNames(dataDir string) error {
	namesPath := filepath.Join(dataDir, "names")
	surnamesPath := filepath.Join(dataDir, "surnames")
	if err := names.LoadNameFiles(namesPath, surnamesPath); err != nil {
		return fmt.Errorf("load embedded names override: %w", err)
	}

	return nil
}

func toSessionConfig(cfg config) session.Config {
	return session.Config{
		Label:           cfg.label,
		Mode:            cfg.mode,
		Link:            cfg.link,
		Transport:       cfg.transport,
		Carrier:         cfg.carrier,
		RoomID:          cfg.roomID,
		ClientID:        cfg.clientID,
		KeyHex:          cfg.keyHex,
		SOCKSHost:       cfg.socksHost,
		SOCKSPort:       cfg.socksPort,
		DNSServer:       cfg.dnsServer,
		SOCKSProxyAddr:  cfg.socksProxyAddr,
		SOCKSProxyPort:  cfg.socksProxyPort,
		VideoWidth:      cfg.videoWidth,
		VideoHeight:     cfg.videoHeight,
		VideoFPS:        cfg.videoFPS,
		VideoBitrate:    cfg.videoBitrate,
		VideoHW:         cfg.videoHW,
		VideoQRSize:     cfg.videoQRSize,
		VideoQRRecovery: cfg.videoQRRecovery,
		VideoCodec:      cfg.videoCodec,
		VideoTileModule: cfg.videoTileModule,
		VideoTileRS:     cfg.videoTileRS,
		VP8FPS:          cfg.vp8FPS,
		VP8BatchSize:    cfg.vp8BatchSize,
		SEIFPS:          cfg.seiFPS,
		SEIBatchSize:    cfg.seiBatchSize,
		SEIFragmentSize: cfg.seiFragmentSize,
		SEIAckTimeoutMS: cfg.seiAckTimeoutMS,
		Lifetime:        cfg.lifetime,
	}
}

func waitForShutdown(errCh <-chan error, count int) error {
	done := make(chan error, 1)
	go func() {
		var firstErr error
		for i := 0; i < count; i++ {
			if err := <-errCh; err != nil && firstErr == nil {
				firstErr = err
			}
		}
		done <- firstErr
	}()

	select {
	case err := <-done:
		if err == nil {
			logger.Info("Shutdown complete")
		}
		return err
	case <-time.After(5 * time.Second):
		logger.Warn("Shutdown timeout, forcing exit")
		return nil
	}
}
