package scanner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Fischris/paperless-scanner/internal/configuration"
)

type Service struct {
	scannerConfiguration configuration.ScannerConfiguration
	scanSemaphore        chan struct{}
}

func NewService(scannerConfiguration configuration.ScannerConfiguration) *Service {
	return &Service{
		scannerConfiguration: scannerConfiguration,
		scanSemaphore:        make(chan struct{}, 1),
	}
}

func (scannerService *Service) DiscoverScanners() error {
	log.Printf("discovering scanners with scanimage -L")

	discoveryCommand := exec.Command("scanimage", "-L")
	discoveryOutput, discoveryCommandError := discoveryCommand.CombinedOutput()

	if len(discoveryOutput) > 0 {
		log.Printf("scanimage -L output:\n%s", strings.TrimSpace(string(discoveryOutput)))
	}

	if discoveryCommandError != nil {
		return fmt.Errorf("scanimage -L failed: %w", discoveryCommandError)
	}

	return nil
}

func (scannerService *Service) TryAcquireScanSlot() bool {
	select {
	case scannerService.scanSemaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

func (scannerService *Service) ReleaseScanSlot() {
	select {
	case <-scannerService.scanSemaphore:
	default:
		log.Printf("scan slot release called without active scan")
	}
}

func (scannerService *Service) RunFlatbedScan() error {
	timestamp := time.Now().Format("20060102_150405")
	outputFilePath := filepath.Join(
		scannerService.scannerConfiguration.TargetDirectory,
		fmt.Sprintf("scan_flatbed_%s.pdf", timestamp),
	)

	scanArguments := []string{
		"--source", "Flatbed",
		"--format=pdf",
		"--resolution", scannerService.scannerConfiguration.ScanResolutionDPI,
	}

	if scannerService.scannerConfiguration.ScannerDevice != "" {
		scanArguments = append(
			[]string{"-d", scannerService.scannerConfiguration.ScannerDevice},
			scanArguments...,
		)
	}

	commandContext, cancelCommandContext := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelCommandContext()

	scanCommand := exec.CommandContext(commandContext, "scanimage", scanArguments...)
	scanCommand.Stderr = os.Stderr

	outputFile, outputFileError := os.Create(outputFilePath)
	if outputFileError != nil {
		return fmt.Errorf("create output file: %w", outputFileError)
	}

	scanCommand.Stdout = outputFile

	scanCommandError := scanCommand.Run()

	outputFileCloseError := outputFile.Close()
	if outputFileCloseError != nil {
		log.Printf("failed to close output file: %v", outputFileCloseError)
	}

	if scanCommandError != nil {
		outputFileRemoveError := os.Remove(outputFilePath)
		if outputFileRemoveError != nil && !errors.Is(outputFileRemoveError, os.ErrNotExist) {
			log.Printf("failed to remove output file after scan error: %v", outputFileRemoveError)
		}
		return fmt.Errorf("scanimage failed: %w", scanCommandError)
	}

	outputFileInfo, outputFileStatError := os.Stat(outputFilePath)
	if outputFileStatError != nil {
		outputFileRemoveError := os.Remove(outputFilePath)
		if outputFileRemoveError != nil && !errors.Is(outputFileRemoveError, os.ErrNotExist) {
			log.Printf("failed to remove output file after stat error: %v", outputFileRemoveError)
		}
		return fmt.Errorf("stat output file: %w", outputFileStatError)
	}

	if outputFileInfo.Size() == 0 {
		outputFileRemoveError := os.Remove(outputFilePath)
		if outputFileRemoveError != nil && !errors.Is(outputFileRemoveError, os.ErrNotExist) {
			log.Printf("failed to remove empty output file: %v", outputFileRemoveError)
		}
		return errors.New("output file is empty")
	}

	return nil
}

func (scannerService *Service) RunADFScan() error {
	timestamp := time.Now().Format("20060102_150405")
	outputFilePattern := filepath.Join(
		scannerService.scannerConfiguration.TargetDirectory,
		fmt.Sprintf("scan_adf_%s_%%d.pdf", timestamp),
	)

	scanArguments := []string{
		"--source", "ADF",
		"--format=pdf",
		"--resolution", scannerService.scannerConfiguration.ScanResolutionDPI,
		"--batch=" + outputFilePattern,
	}

	if scannerService.scannerConfiguration.ScannerDevice != "" {
		scanArguments = append(
			[]string{"-d", scannerService.scannerConfiguration.ScannerDevice},
			scanArguments...,
		)
	}

	commandContext, cancelCommandContext := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelCommandContext()

	scanCommand := exec.CommandContext(commandContext, "scanimage", scanArguments...)
	scanCommand.Stderr = os.Stderr

	scanCommandError := scanCommand.Run()
	if scanCommandError != nil {
		return fmt.Errorf("scanimage failed: %w", scanCommandError)
	}

	outputFileMatches, outputFileMatchError := filepath.Glob(
		filepath.Join(
			scannerService.scannerConfiguration.TargetDirectory,
			fmt.Sprintf("scan_adf_%s_*.pdf", timestamp),
		),
	)
	if outputFileMatchError != nil {
		return fmt.Errorf("list output files: %w", outputFileMatchError)
	}
	if len(outputFileMatches) == 0 {
		return errors.New("no output files created")
	}

	hasNonEmptyOutputFile := false
	for _, outputFilePath := range outputFileMatches {
		outputFileInfo, outputFileStatError := os.Stat(outputFilePath)
		if outputFileStatError != nil {
			return fmt.Errorf("stat output file: %w", outputFileStatError)
		}
		if outputFileInfo.Size() > 0 {
			hasNonEmptyOutputFile = true
			break
		}
	}

	if !hasNonEmptyOutputFile {
		return errors.New("all output files are empty")
	}

	return nil
}
