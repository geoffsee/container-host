package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ulikunitz/xz"
)

// progressWriter tracks and prints progress (download or extract).
type progressWriter struct {
	label     string
	total     int64 // -1 if unknown
	processed int64
	last      time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.processed += int64(n)
	now := time.Now()
	if pw.last.IsZero() || now.Sub(pw.last) >= 200*time.Millisecond || (pw.total > 0 && pw.processed >= pw.total) {
		if pw.total > 0 {
			percent := float64(pw.processed) / float64(pw.total) * 100
			fmt.Printf("\r%s... %s/%s (%.1f%%)", pw.label, humanizeBytes(pw.processed), humanizeBytes(pw.total), percent)
		} else {
			fmt.Printf("\r%s... %s", pw.label, humanizeBytes(pw.processed))
		}
		pw.last = now
	}
	return n, nil
}

func humanizeBytes(b int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	f := float64(b)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

// ensureCoreOSImage ensures the Fedora CoreOS image exists in vms/,
// downloads if missing, links into images/, and extracts a .qcow2 if needed.
func ensureCoreOSImage(projectRoot, version, arch string) error {
	vmsDir := filepath.Join(projectRoot, "vms")
	imagesDir := filepath.Join(projectRoot, "images")

	if err := os.MkdirAll(vmsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create vms dir: %w", err)
	}
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return fmt.Errorf("failed to create images dir: %w", err)
	}

	vmsFile := filepath.Join(vmsDir, fmt.Sprintf("coreos-%s-%s.xz", version, arch))
	imagesFile := filepath.Join(imagesDir, fmt.Sprintf("coreos-%s-%s.xz", version, arch))
	qcow2File := filepath.Join(imagesDir, fmt.Sprintf("coreos-%s-qemu.%s.qcow2", version, arch))

	url := fmt.Sprintf(
		"https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/%s/%s/fedora-coreos-%s-qemu.%s.qcow2.xz",
		version, arch, version, arch,
	)

	// Download if not cached
	if _, err := os.Stat(vmsFile); os.IsNotExist(err) {
		fmt.Printf("Downloading Fedora CoreOS image from %s...\n", url)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download image: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad HTTP status: %s", resp.Status)
		}

		out, err := os.Create(vmsFile)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer out.Close()

		pw := &progressWriter{label: "Downloading", total: resp.ContentLength}
		if _, err := io.Copy(out, io.TeeReader(resp.Body, pw)); err != nil {
			fmt.Print("\n")
			return fmt.Errorf("failed to save image: %w", err)
		}
		fmt.Print("\n")
		fmt.Printf("Downloaded to: %s\n", vmsFile)
	} else {
		fmt.Printf("Image already exists: %s\n", vmsFile)
	}

	// Hard-link .xz into images/ (fallback to copy if cross-device)
	if err := linkOrCopy(vmsFile, imagesFile); err != nil {
		return fmt.Errorf("failed to place image in images/: %w", err)
	}
	fmt.Printf("Placed compressed image at: %s\n", imagesFile)

	// Extract into .qcow2 if needed
	if _, err := os.Stat(qcow2File); os.IsNotExist(err) {
		fmt.Printf("Extracting %s → %s ...\n", vmsFile, qcow2File)
		if err := extractXZFast(vmsFile, qcow2File); err != nil {
			return fmt.Errorf("failed to extract image: %w", err)
		}
		fmt.Printf("Extracted to: %s\n", qcow2File)
	} else {
		fmt.Printf("Extracted image already exists: %s\n", qcow2File)
	}

	return nil
}

// linkOrCopy tries to hard-link src→dst, falls back to copy if needed.
func linkOrCopy(src, dst string) error {
	if fi, err := os.Stat(dst); err == nil {
		if sfi, err2 := os.Stat(src); err2 == nil && fi.Size() == sfi.Size() {
			return nil
		}
		_ = os.Remove(dst)
	}
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	return copyFile(src, dst)
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer src.Close()
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

// extractXZFast prefers native xz/unxz for speed, falls back to pure-Go xz.
func extractXZFast(srcPath, dstPath string) error {
	if xzPath, err := exec.LookPath("xz"); err == nil {
		if err := extractWithExternalXZ(xzPath, srcPath, dstPath); err == nil {
			return nil
		}
	}
	if unxzPath, err := exec.LookPath("unxz"); err == nil {
		if err := extractWithExternalXZ(unxzPath, srcPath, dstPath); err == nil {
			return nil
		}
	}
	return extractWithGoXZ(srcPath, dstPath)
}

func extractWithExternalXZ(bin, srcPath, dstPath string) error {
	cmd := exec.Command(bin, "-T0", "-dc", srcPath) // multithreaded
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start xz: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dstPath), filepath.Base(dstPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer func() { tmp.Close(); _ = os.Remove(tmp.Name()) }()

	pw := &progressWriter{label: "Extracting", total: -1}
	if _, err := io.Copy(io.MultiWriter(tmp, pw), stdout); err != nil {
		slurp, _ := io.ReadAll(stderr)
		_ = cmd.Wait()
		fmt.Print("\n")
		return fmt.Errorf("xz copy failed: %w; stderr=%s", err, string(slurp))
	}
	if err := cmd.Wait(); err != nil {
		slurp, _ := io.ReadAll(stderr)
		fmt.Print("\n")
		return fmt.Errorf("xz failed: %w; stderr=%s", err, string(slurp))
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("fsync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmp.Name(), dstPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	fmt.Print("\n")
	return nil
}

func extractWithGoXZ(srcPath, dstPath string) error {
	in, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	br := bufio.NewReaderSize(in, 1<<20)
	xzr, err := xz.NewReader(br)
	if err != nil {
		return fmt.Errorf("xz reader: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dstPath), filepath.Base(dstPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer func() { tmp.Close(); _ = os.Remove(tmp.Name()) }()

	pw := &progressWriter{label: "Extracting", total: -1}
	buf := make([]byte, 1<<20)
	if _, err := io.CopyBuffer(io.MultiWriter(tmp, pw), xzr, buf); err != nil {
		fmt.Print("\n")
		return fmt.Errorf("decompress copy: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("fsync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmp.Name(), dstPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	fmt.Print("\n")
	return nil
}
