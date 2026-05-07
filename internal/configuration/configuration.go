package configuration

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type ScannerConfiguration struct {
	TargetDirectory   string
	AuthToken         string
	ScannerDevice     string
	ScanResolutionDPI string
	ListenPort        string
	RotateOddPages180 bool
}

func LoadScannerConfiguration() (ScannerConfiguration, error) {
	rotateOddPages180Value := strings.ToLower(strings.TrimSpace(os.Getenv("ROTATE_ODD_PAGES_180")))

	scannerConfiguration := ScannerConfiguration{
		TargetDirectory:   strings.TrimSpace(os.Getenv("TARGET_DIR")),
		AuthToken:         strings.TrimSpace(os.Getenv("AUTH_TOKEN")),
		ScannerDevice:     strings.TrimSpace(os.Getenv("SCANNER_DEVICE")),
		ScanResolutionDPI: strings.TrimSpace(os.Getenv("SCAN_RESOLUTION")),
		ListenPort:        strings.TrimSpace(os.Getenv("PORT")),
		RotateOddPages180: rotateOddPages180Value == "1" || rotateOddPages180Value == "true" || rotateOddPages180Value == "yes",
	}

	if scannerConfiguration.TargetDirectory == "" {
		return ScannerConfiguration{}, errors.New("TARGET_DIR is required")
	}
	if scannerConfiguration.AuthToken == "" {
		return ScannerConfiguration{}, errors.New("AUTH_TOKEN is required")
	}
	if scannerConfiguration.ScanResolutionDPI == "" {
		scannerConfiguration.ScanResolutionDPI = "300"
	}
	if scannerConfiguration.ListenPort == "" {
		scannerConfiguration.ListenPort = "8080"
	}

	targetDirectoryInfo, targetDirectoryError := os.Stat(scannerConfiguration.TargetDirectory)
	if targetDirectoryError != nil {
		return ScannerConfiguration{}, fmt.Errorf("invalid TARGET_DIR: %w", targetDirectoryError)
	}
	if !targetDirectoryInfo.IsDir() {
		return ScannerConfiguration{}, errors.New("TARGET_DIR must be a directory")
	}

	testFile, testFileError := os.CreateTemp(scannerConfiguration.TargetDirectory, ".write-test-*")
	if testFileError != nil {
		return ScannerConfiguration{}, fmt.Errorf("TARGET_DIR is not writable: %w", testFileError)
	}

	testFilePath := testFile.Name()

	testFileCloseError := testFile.Close()
	if testFileCloseError != nil {
		return ScannerConfiguration{}, fmt.Errorf("failed to close test file: %w", testFileCloseError)
	}

	testFileRemoveError := os.Remove(testFilePath)
	if testFileRemoveError != nil {
		return ScannerConfiguration{}, fmt.Errorf("failed to remove test file: %w", testFileRemoveError)
	}

	return scannerConfiguration, nil
}
