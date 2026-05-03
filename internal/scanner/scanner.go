package scanner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	return scannerService.runADFScanToSinglePDF("ADF")
}

func (scannerService *Service) RunADFDuplexScan() error {
	return scannerService.runADFScanToSinglePDF("ADF Duplex")
}

// runADFScanToSinglePDF scans pages into a temp directory as PNG and merges them into a single PDF.
func (scannerService *Service) runADFScanToSinglePDF(source string) error {
	timestamp := time.Now().Format("20060102_150405")

	tmpDir, tmpDirError := os.MkdirTemp("", "paperless-scanner-adf-*")
	if tmpDirError != nil {
		return fmt.Errorf("create temp dir: %w", tmpDirError)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	pagePattern := filepath.Join(tmpDir, "page-%04d.png")
	outputFilePath := filepath.Join(
		scannerService.scannerConfiguration.TargetDirectory,
		fmt.Sprintf("scan_adf_%s.pdf", timestamp),
	)

	//scan pages via scanimage  to PNG files
	scanArguments := []string{
		"--source", source,
		"--format=png",
		"--resolution", scannerService.scannerConfiguration.ScanResolutionDPI,
		"--batch=" + pagePattern,
	}

	if scannerService.scannerConfiguration.ScannerDevice != "" {
		scanArguments = append([]string{"-d", scannerService.scannerConfiguration.ScannerDevice}, scanArguments...)
	}

	commandContext, cancelCommandContext := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelCommandContext()

	scanCommand := exec.CommandContext(commandContext, "scanimage", scanArguments...)
	scanCommand.Stderr = os.Stderr

	if err := scanCommand.Run(); err != nil {
		return fmt.Errorf("scanimage failed: %w", err)
	}

	pageFiles, globErr := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if globErr != nil {
		return fmt.Errorf("list scanned pages: %w", globErr)
	}
	if len(pageFiles) == 0 {
		return errors.New("no pages scanned")
	}
	sort.Strings(pageFiles)

	for _, p := range pageFiles {
		info, statErr := os.Stat(p)
		if statErr != nil {
			return fmt.Errorf("stat scanned page: %w", statErr)
		}
		if info.Size() == 0 {
			return fmt.Errorf("scanned page is empty: %s", p)
		}
	}

	//Merge into one PDF via img2pdf
	img2pdfArgs := append([]string{"-o", outputFilePath}, pageFiles...)
	mergeCmd := exec.Command("img2pdf", img2pdfArgs...)
	mergeCmd.Stderr = os.Stderr

	if err := mergeCmd.Run(); err != nil {
		_ = os.Remove(outputFilePath)
		return fmt.Errorf("img2pdf failed: %w", err)
	}

	outInfo, outStatErr := os.Stat(outputFilePath)
	if outStatErr != nil {
		_ = os.Remove(outputFilePath)
		return fmt.Errorf("stat output pdf: %w", outStatErr)
	}
	if outInfo.Size() == 0 {
		_ = os.Remove(outputFilePath)
		return errors.New("output pdf is empty")
	}

	return nil
}
