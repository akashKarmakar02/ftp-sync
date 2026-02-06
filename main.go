package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jlaffaye/ftp"
)

func main() {
	// 1. Configuration from environment variables
	ftpHost := os.Getenv("FTP_HOST")
	if ftpHost == "" {
		ftpHost = "localhost:21"
	}
	ftpUser := os.Getenv("FTP_USER")
	ftpPass := os.Getenv("FTP_PASS")

	remoteDir := os.Getenv("REMOTE_DIR")
	if remoteDir == "" {
		remoteDir = "/"
	}

	localDir := os.Getenv("LOCAL_DIR")
	if localDir == "" {
		localDir = "./sync_dir"
	}

	fmt.Printf("Connecting to %s as %s...\n", ftpHost, ftpUser)

	// 2. Connect to FTP
	c, err := ftp.Dial(ftpHost, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		log.Fatalf("Error connecting to FTP: %v", err)
	}

	if err := c.Login(ftpUser, ftpPass); err != nil {
		log.Fatalf("Error logging in: %v", err)
	}
	defer c.Quit()

	fmt.Println("Connected. Starting sync...")

	// 3. Create local root directory
	if err := os.MkdirAll(localDir, 0755); err != nil {
		log.Fatalf("Error creating local directory: %v", err)
	}

	// 4. Start recursive sync
	if err := syncDir(c, remoteDir, localDir); err != nil {
		log.Fatalf("Error syncing: %v", err)
	}

	fmt.Println("Sync completed successfully!")
}

// syncDir recursively synchronizes the remote directory to the local directory
func syncDir(c *ftp.ServerConn, remotePath, localPath string) error {
	entries, err := c.List(remotePath)
	if err != nil {
		return fmt.Errorf("failed to list %s: %w", remotePath, err)
	}

	for _, entry := range entries {
		// Skip . and ..
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		// Construct paths
		// FTP paths use forward slash
		currentRemotePath := path.Join(remotePath, entry.Name)
		// Local paths use OS separator
		currentLocalPath := filepath.Join(localPath, entry.Name)

		if entry.Type == ftp.EntryTypeFolder {
			// Directory: Create local dir and recurse
			if err := os.MkdirAll(currentLocalPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", currentLocalPath, err)
			}
			// fmt.Printf("Dir: %s -> %s\n", currentRemotePath, currentLocalPath)
			if err := syncDir(c, currentRemotePath, currentLocalPath); err != nil {
				return err
			}
		} else {
			// File: Download
			fmt.Printf("Downloading: %s\n", currentRemotePath)
			if err := downloadFile(c, currentRemotePath, currentLocalPath); err != nil {
				return fmt.Errorf("failed to download %s: %w", currentRemotePath, err)
			}
		}
	}
	return nil
}

func downloadFile(c *ftp.ServerConn, remoteFile, localFile string) error {
	resp, err := c.Retr(remoteFile)
	if err != nil {
		return err
	}
	defer resp.Close()

	outFile, err := os.Create(localFile)
	if err != nil {
		return err
	}
	// Ensure file is closed even if copy fails, usually handled by defer but
	// in a loop/function it's safer to be explicit or use a closure if we were tight on resources.
	// Since this is a separate function, defer is fine.
	defer outFile.Close()

	_, err = io.Copy(outFile, resp)
	return err
}
