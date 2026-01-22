package download

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	maxRetries      = 3
	retryDelay      = 2 * time.Second
	downloadTimeout = 30 * time.Minute
)

// DownloadWithProgress downloads a file with progress reporting
func DownloadWithProgress(
	dest string,
	url string,
	stage string,
	progressWeight float64,
	callback func(stage string, progress float64, message string, currentFile string, speed string, downloaded, total int64),
) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := attemptDownload(dest, url, stage, progressWeight, callback)
		if err == nil {
			return nil
		}

		lastErr = err
		fmt.Printf("Download attempt %d failed: %v\n", attempt, err)

		// If certificate error and trusted source (github/adoptium), try with insecure client
		if attempt == 1 && isCertError(err) && isTrustedSource(url) {
			fmt.Println("Certificate verification failed, retrying with insecure client for trusted source...")
			err = attemptDownloadInsecure(dest, url, stage, progressWeight, callback)
			if err == nil {
				fmt.Println("Download successful with insecure client")
				return nil
			}
			lastErr = err
		}

		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("download failed after %d attempts: %w", maxRetries, lastErr)
}

// isCertError checks if error is TLS certificate related
func isCertError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "certificate") || contains(errStr, "x509") || contains(errStr, "tls")
}

// isTrustedSource checks if URL is from trusted sources
func isTrustedSource(url string) bool {
	trustedDomains := []string{
		"github.com",
		"githubusercontent.com",
		"adoptium.net",
		"itch.zone",
	}
	for _, domain := range trustedDomains {
		if contains(url, domain) {
			return true
		}
	}
	return false
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func attemptDownload(
	dest string,
	url string,
	stage string,
	progressWeight float64,
	callback func(stage string, progress float64, message string, currentFile string, speed string, downloaded, total int64),
) error {
	client := createOptimizedClient()

	tempDest := dest + ".tmp"

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if partial file exists
	var resumeFrom int64 = 0
	if stat, err := os.Stat(tempDest); err == nil {
		resumeFrom = stat.Size()
	}

	// Create request with context for timeout control
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "HyPrism/1.0")

	if resumeFrom > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Handle resume
	var file *os.File
	if resp.StatusCode == http.StatusPartialContent {
		file, err = os.OpenFile(tempDest, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(tempDest)
		resumeFrom = 0
	}
	if err != nil {
		return err
	}
	defer file.Close()

	total := resp.ContentLength + resumeFrom
	downloaded := resumeFrom

	buf := make([]byte, 32*1024)
	lastUpdate := time.Now()
	lastDownloaded := downloaded

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			// Update progress every 100ms
			if time.Since(lastUpdate) >= 100*time.Millisecond && callback != nil {
				speed := float64(downloaded-lastDownloaded) / time.Since(lastUpdate).Seconds()
				speedStr := formatSpeed(speed)
				progress := float64(downloaded) / float64(total) * 100 * progressWeight

				callback(stage, progress, "Downloading...", filepath.Base(dest), speedStr, downloaded, total)

				lastUpdate = time.Now()
				lastDownloaded = downloaded
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	file.Close()

	// Rename temp file to final destination
	if err := os.Rename(tempDest, dest); err != nil {
		return err
	}

	return nil
}

// attemptDownloadInsecure is identical to attemptDownload but uses insecure client
func attemptDownloadInsecure(
	dest string,
	url string,
	stage string,
	progressWeight float64,
	callback func(stage string, progress float64, message string, currentFile string, speed string, downloaded, total int64),
) error {
	client := insecureClient

	tempDest := dest + ".tmp"

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if partial file exists
	var resumeFrom int64 = 0
	if stat, err := os.Stat(tempDest); err == nil {
		resumeFrom = stat.Size()
	}

	// Create request with context for timeout control
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "HyPrism/1.0")

	if resumeFrom > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Handle resume
	var file *os.File
	if resp.StatusCode == http.StatusPartialContent {
		file, err = os.OpenFile(tempDest, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(tempDest)
		resumeFrom = 0
	}
	if err != nil {
		return err
	}
	defer file.Close()

	total := resp.ContentLength + resumeFrom
	downloaded := resumeFrom

	buf := make([]byte, 32*1024)
	lastUpdate := time.Now()
	lastDownloaded := downloaded

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			// Update progress every 100ms
			if time.Since(lastUpdate) >= 100*time.Millisecond && callback != nil {
				speed := float64(downloaded-lastDownloaded) / time.Since(lastUpdate).Seconds()
				speedStr := formatSpeed(speed)
				progress := float64(downloaded) / float64(total) * 100 * progressWeight

				callback(stage, progress, "Downloading...", filepath.Base(dest), speedStr, downloaded, total)

				lastUpdate = time.Now()
				lastDownloaded = downloaded
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	file.Close()

	// Rename temp file to final destination
	if err := os.Rename(tempDest, dest); err != nil {
		return err
	}

	return nil
}

var (
	defaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableCompression:    true,
	}

	// Transport with insecure TLS for trusted sources (JRE downloads from GitHub/Adoptium)
	insecureTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, // Skip cert verification for systems with broken cert stores
		},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableCompression:    true,
	}

	// sharedClient is a singleton HTTP client used to enable TCP connection reuse (Keep-Alive)
	// across different parts of the application, reducing handshake overhead.
	sharedClient = &http.Client{
		Transport: defaultTransport,
		Timeout:   downloadTimeout,
	}

	// insecureClient for trusted sources when cert verification fails
	insecureClient = &http.Client{
		Transport: insecureTransport,
		Timeout:   downloadTimeout,
	}
)

// GetSharedClient returns a globally shared optimized HTTP client
func GetSharedClient() *http.Client {
	return sharedClient
}

func createOptimizedClient() *http.Client {
	return sharedClient
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	} else {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
	}
}

// CreateTempFile creates a temporary file
func CreateTempFile(pattern string) (string, error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	file.Close()
	return file.Name(), nil
}

// DownloadLatestReleaseAsset downloads an asset from the latest stable GitHub release
func DownloadLatestReleaseAsset(ctx context.Context, assetName, dest string, callback func(stage string, progress float64, message string, currentFile string, speed string, downloaded, total int64)) error {
	return DownloadReleaseAsset(ctx, assetName, dest, false, callback)
}

// DownloadReleaseAsset downloads an asset from either stable release or nightly pre-release
func DownloadReleaseAsset(ctx context.Context, assetName, dest string, isNightly bool, callback func(stage string, progress float64, message string, currentFile string, speed string, downloaded, total int64)) error {
	owner := "yyyumeniku"
	repo := "HyPrism"
	
	var url string
	if isNightly {
		// For nightly builds, get from the latest pre-release (tagged as nightly)
		url = fmt.Sprintf("https://github.com/%s/%s/releases/download/nightly/%s", owner, repo, assetName)
	} else {
		// For stable releases, get from /releases/latest
		url = fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/%s", owner, repo, assetName)
	}
	
	return DownloadWithProgress(dest, url, "download", 1.0, callback)
}

// GetSystemArch returns the system architecture in a normalized format
func GetSystemArch() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		return "x64"
	}
	return arch
}

// DownloadFile downloads a file with a simple progress callback
func DownloadFile(ctx context.Context, url, dest string, progressCallback func(downloaded, total int64, speed string)) error {
	client := createOptimizedClient()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	startTime := time.Now()
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			if progressCallback != nil {
				elapsed := time.Since(startTime).Seconds()
				speed := ""
				if elapsed > 0 {
					speed = formatSpeed(float64(downloaded) / elapsed)
				}
				progressCallback(downloaded, total, speed)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}
