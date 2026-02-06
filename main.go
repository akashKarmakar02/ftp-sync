package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/jlaffaye/ftp"
)

func main() {
	srcHost := os.Getenv("FTP_HOST")
	if srcHost == "" {
		srcHost = "localhost:21"
	}
	srcUser := os.Getenv("FTP_USER")
	srcPass := os.Getenv("FTP_PASS")

	destHost := os.Getenv("DEST_FTP_HOST")
	if destHost == "" {
		destHost = "localhost:21"
	}
	destUser := os.Getenv("DEST_FTP_USER")
	destPass := os.Getenv("DEST_FTP_PASS")

	srcDir := os.Getenv("REMOTE_DIR")
	if srcDir == "" {
		srcDir = ""
	}

	destDir := os.Getenv("DEST_DIR")
	if destDir == "" {
		destDir = ""
	}

	fmt.Printf("Connecting to Source %s as %s...\n", srcHost, srcUser)
	srcConn, err := connectFTP(srcHost, srcUser, srcPass)
	if err != nil {
		log.Fatalf("Error connecting to source FTP: %v", err)
	}
	defer srcConn.Quit()

	fmt.Printf("Connecting to Destination %s as %s...\n", destHost, destUser)
	destConn, err := connectFTP(destHost, destUser, destPass)
	if err != nil {
		log.Fatalf("Error connecting to destination FTP: %v", err)
	}
	defer destConn.Quit()

	fmt.Println("Connected to both servers. Starting sync...")

	if err := srcConn.ChangeDir(srcDir); err != nil {
		log.Fatalf("Source directory %s not accessible: %v", srcDir, err)
	}

	if err := syncDir(srcConn, destConn, srcDir, destDir); err != nil {
		log.Fatalf("Error syncing: %v", err)
	}

	fmt.Println("Sync completed successfully!")
}

func connectFTP(host, user, pass string) (*ftp.ServerConn, error) {
	c, err := ftp.Dial(host, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return nil, err
	}

	if err := c.Login(user, pass); err != nil {
		c.Quit()
		return nil, err
	}
	return c, nil
}

// syncDir recursively synchronizes the source FTP directory to the destination FTP directory
func syncDir(src, dest *ftp.ServerConn, srcPath, destPath string) error {
	entries, err := src.List(srcPath)
	if err != nil {
		return fmt.Errorf("failed to list %s on source: %w", srcPath, err)
	}

	if destPath != "." && destPath != "/" {
		if err := dest.MakeDir(destPath); err != nil {
			// Ignore error if dir likely exists.
			// Unfortunately ftp lib doesn't return typed errors easily to distinguish EEXIST.
			// Usually 550.
		}
	}

	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		fmt.Printf("Processing %s\n", entry.Name)

		nextSrcPath := path.Join(srcPath, entry.Name)
		nextDestPath := path.Join(destPath, entry.Name)

		if entry.Type == ftp.EntryTypeFolder {
			if err := syncDir(src, dest, nextSrcPath, nextDestPath); err != nil {
				return err
			}
		} else {
			fmt.Printf("Transferring: %s -> %s\n", nextSrcPath, nextDestPath)
			if err := transferFile(src, dest, nextSrcPath, nextDestPath); err != nil {
				return fmt.Errorf("failed to transfer %s: %w", nextSrcPath, err)
			}
		}
	}
	return nil
}

func transferFile(src, dest *ftp.ServerConn, srcFile, destFile string) error {
	resp, err := src.Retr(srcFile)
	if err != nil {
		return fmt.Errorf("source RETR failed: %w", err)
	}
	defer resp.Close()

	if err := dest.Stor(destFile, resp); err != nil {
		return fmt.Errorf("destination STOR failed: %w", err)
	}

	return nil
}
