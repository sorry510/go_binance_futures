package historicalmarket

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Only BT-3 verified ZIPs have a .sha256 sidecar. Other files in the cache
// directory are never considered for pruning.
var activePublicArchives = struct {
	sync.Mutex
	paths map[string]int
}{paths: make(map[string]int)}

func (client *PublicDataClient) archiveTarget(spec PublicDataArchiveSpec, archiveURL string) (string, error) {
	_, filename, err := publicDataArchivePath(spec)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(archiveURL))
	return filepath.Join(client.cacheDir, hex.EncodeToString(key[:8])+"-"+filename), nil
}

// Register before inspecting a disk entry so a concurrently finishing run
// in this process cannot evict a file being loaded.
func (client *PublicDataClient) holdArchive(target string) {
	if client.maxCacheBytes <= 0 {
		return
	}
	client.mu.Lock()
	if _, ok := client.heldArchives[target]; !ok {
		activePublicArchives.Lock()
		activePublicArchives.paths[target]++
		activePublicArchives.Unlock()
		client.heldArchives[target] = struct{}{}
	}
	client.mu.Unlock()
}

func (client *PublicDataClient) releaseArchives() {
	if client.maxCacheBytes <= 0 {
		return
	}
	client.mu.Lock()
	activePublicArchives.Lock()
	for target := range client.heldArchives {
		activePublicArchives.paths[target]--
		if activePublicArchives.paths[target] <= 0 {
			delete(activePublicArchives.paths, target)
		}
	}
	client.heldArchives = make(map[string]struct{})
	activePublicArchives.Unlock()
	client.mu.Unlock()
}

func (client *PublicDataClient) loadDiskArchive(spec PublicDataArchiveSpec, archiveURL string) (PublicDataArchive, bool) {
	target, err := client.archiveTarget(spec, archiveURL)
	if err != nil {
		return PublicDataArchive{}, false
	}
	client.holdArchive(target)
	meta, err := os.ReadFile(target + ".sha256")
	if err != nil {
		return PublicDataArchive{}, false
	}
	fields := strings.Fields(string(meta))
	if len(fields) != 2 || !validSHA256Hex(fields[0]) {
		return PublicDataArchive{}, false
	}
	wantSize, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || wantSize <= 0 {
		return PublicDataArchive{}, false
	}
	file, err := os.Open(target)
	if err != nil {
		return PublicDataArchive{}, false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != wantSize {
		return PublicDataArchive{}, false
	}
	hash := sha256.New()
	read, err := io.Copy(hash, file)
	if err != nil || read != wantSize || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), fields[0]) {
		return PublicDataArchive{}, false
	}
	now := time.Now()
	_ = os.Chtimes(target, now, now) // LRU = last successful use
	return PublicDataArchive{
		Spec: spec, URL: archiveURL, ChecksumURL: archiveURL + ".CHECKSUM",
		Path: target, SHA256: strings.ToLower(fields[0]), Size: wantSize, FromCache: true,
	}, true
}

func (client *PublicDataClient) saveArchiveManifest(target, sha string, size int64) error {
	file, err := os.CreateTemp(client.cacheDir, ".bt3-manifest-*.part")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	if _, err = fmt.Fprintf(file, "%s %d\n", sha, size); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, target+".sha256")
}

// pruneVerifiedArchives implements a byte-budget LRU. Only completed verified
// archive pairs are considered; concurrently active clients in this process
// retain their files until they close. Pruning is best effort for disk I/O.
func (client *PublicDataClient) pruneVerifiedArchives() error {
	if client.maxCacheBytes <= 0 || client.ownsCacheDir {
		return nil
	}
	entries, err := os.ReadDir(client.cacheDir)
	if err != nil {
		return err
	}
	type entry struct {
		path string
		mod  time.Time
		size int64
	}
	var list []entry
	var total int64
	for _, item := range entries {
		if item.IsDir() {
			continue
		}
		// Aborted downloads leave only BT-3-owned temporary files. Remove
		// abandoned fragments after one day, never in-progress fragments.
		if strings.HasPrefix(item.Name(), ".bt3-download-") || strings.HasPrefix(item.Name(), ".bt3-manifest-") {
			if info, statErr := item.Info(); statErr == nil && time.Since(info.ModTime()) > 24*time.Hour {
				_ = os.Remove(filepath.Join(client.cacheDir, item.Name()))
			}
			continue
		}
		if !strings.HasSuffix(item.Name(), ".zip.sha256") {
			continue
		}
		name := strings.TrimSuffix(item.Name(), ".sha256")
		// Protect user files; the prefix is the first 8 bytes of SHA256(URL).
		if len(name) < 21 || name[16] != '-' {
			continue
		}
		if _, err := hex.DecodeString(name[:16]); err != nil {
			continue
		}
		target := filepath.Join(client.cacheDir, name)
		info, err := os.Lstat(target)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		metadata, err := os.ReadFile(target + ".sha256")
		if err != nil {
			continue
		}
		fields := strings.Fields(string(metadata))
		if len(fields) != 2 || !validSHA256Hex(fields[0]) {
			continue
		}
		size, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || size <= 0 || size != info.Size() {
			continue
		}
		list = append(list, entry{path: target, size: info.Size(), mod: info.ModTime()})
		total += info.Size()
	}
	if total <= client.maxCacheBytes {
		return nil
	}
	sort.Slice(list, func(i, j int) bool { return list[i].mod.Before(list[j].mod) })
	activePublicArchives.Lock()
	defer activePublicArchives.Unlock()
	for _, item := range list {
		if total <= client.maxCacheBytes {
			break
		}
		if activePublicArchives.paths[item.path] > 0 {
			continue
		}
		if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		_ = os.Remove(item.path + ".sha256")
		total -= item.size
	}
	return nil
}
