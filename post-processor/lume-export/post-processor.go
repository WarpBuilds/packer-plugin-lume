// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package lume_export

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// Config defines the post-processor configuration.
type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	VMName              string `mapstructure:"vm_name"`
	Tag                 string `mapstructure:"tag"`
	ChunkSize           string `mapstructure:"chunk_size"`
	ctx                 interpolate.Context
}

func (c Config) FolderPath() string {
	return fmt.Sprintf("~/.lume/%s", c.VMName)
}

// PostProcessor implements the post-processor interface.
type PostProcessor struct {
	config Config
}

// ConfigSpec returns the HCL2 configuration specification.
func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec {
	return p.config.FlatMapstructure().HCL2Spec()
}

// Configure decodes the configuration and sets default values.
func (p *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		PluginType:         "packer.post-processor.lume-export",
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{},
		},
	}, raws...)
	if err != nil {
		return err
	}

	// Set a default chunk size if not provided.
	if p.config.ChunkSize == "" {
		p.config.ChunkSize = "500M"
	}
	return nil
}

// PostProcess processes the artifact and outputs the image tag along with file details.
func (p *PostProcessor) PostProcess(ctx context.Context, ui packersdk.Ui, source packersdk.Artifact) (packersdk.Artifact, bool, bool, error) {
	// Validate configuration.
	if err := p.validateConfig(); err != nil {
		return source, false, false, err
	}

	// Create a temporary working directory.
	workDir, err := ioutil.TempDir(fmt.Sprintf("~/.lume/%s", p.config.VMName), "save-image")
	if err != nil {
		return source, false, false, fmt.Errorf("failed to create work directory: %s", err)
	}
	defer os.RemoveAll(workDir)
	ui.Say(fmt.Sprintf("Working directory: %s", workDir))

	// Copy optional files from the folder path.
	if err := p.copyFileIfExists(filepath.Join(p.config.FolderPath(), "config.json"), filepath.Join(workDir, "config.json"), ui); err != nil {
		return source, false, false, err
	}
	if err := p.copyFileIfExists(filepath.Join(p.config.FolderPath(), "nvram.bin"), filepath.Join(workDir, "nvram.bin"), ui); err != nil {
		return source, false, false, err
	}

	// Process disk image.
	diskImgSrc := filepath.Join(p.config.FolderPath(), "disk.img")
	diskImgPath, err := p.processDiskImg(diskImgSrc, workDir, ui)
	if err != nil {
		return source, false, false, err
	}

	// Build a list of files (for informational purposes).
	files, err := p.buildFilesList(workDir, diskImgPath)
	if err != nil {
		return source, false, false, err
	}

	// Instead of pushing, output the image tag and list the processed files.
	ui.Say(fmt.Sprintf("Image saved with tag: %s", p.config.Tag))
	ui.Say("The following files are part of the image artifact:")
	for _, file := range files {
		ui.Say("  - " + file)
	}

	// Return the original artifact unchanged.
	return source, true, true, nil
}

// validateConfig ensures that required configuration fields are provided.
func (p *PostProcessor) validateConfig() error {
	if p.config.VMName == "" {
		return fmt.Errorf("vm_name is required")
	}
	if p.config.Tag == "" {
		return fmt.Errorf("tag is required")
	}
	return nil
}

// copyFileIfExists copies the source file to the destination if it exists.
func (p *PostProcessor) copyFileIfExists(src, dest string, ui packersdk.Ui) error {
	if _, err := os.Stat(src); err == nil {
		ui.Say(fmt.Sprintf("Copying %s...", filepath.Base(src)))
		input, err := ioutil.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %s", src, err)
		}
		if err = ioutil.WriteFile(dest, input, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %s", dest, err)
		}
	}
	return nil
}

// processDiskImg handles the disk image.
// If disk.img exceeds 500MB it will be split into chunks; otherwise it is simply copied.
func (p *PostProcessor) processDiskImg(srcDiskImg, workDir string, ui packersdk.Ui) (string, error) {
	if _, err := os.Stat(srcDiskImg); err != nil {
		// disk.img does not exist.
		return "", nil
	}
	destDiskImg := filepath.Join(workDir, "disk.img")
	info, err := os.Stat(srcDiskImg)
	if err != nil {
		return "", err
	}
	const sizeThreshold = 524288000 // 500MB in bytes

	if info.Size() > sizeThreshold {
		ui.Say("disk.img is large. Splitting into chunks...")
		// Copy the file first.
		if err := copyFile(srcDiskImg, destDiskImg); err != nil {
			return "", fmt.Errorf("failed to copy disk.img: %s", err)
		}
		// Execute the split command.
		splitPrefix := filepath.Join(workDir, "disk.img.part.")
		splitCmd := exec.Command("split", "-b", p.config.ChunkSize, destDiskImg, splitPrefix)
		if output, err := splitCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to split disk.img: %s, output: %s", err, output)
		}
		// Remove the original large file as it has been split.
		os.Remove(destDiskImg)
		// Return an empty string to indicate that split parts will be used.
		return "", nil
	}

	ui.Say("Copying disk.img...")
	if err := copyFile(srcDiskImg, destDiskImg); err != nil {
		return "", err
	}
	return destDiskImg, nil
}

// copyFile is a helper function to copy a file from src to dest.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// buildFilesList creates a list of files (with annotations) representing the image artifact.
func (p *PostProcessor) buildFilesList(workDir, diskImgPath string) ([]string, error) {
	var files []string

	// If diskImgPath is non-empty, disk.img was small and copied directly.
	if diskImgPath != "" {
		files = append(files, filepath.Join(workDir, "disk.img")+":application/vnd.oci.image.layer.v1.tar")
	} else {
		// Otherwise, look for split parts.
		parts, err := filepath.Glob(filepath.Join(workDir, "disk.img.part.*"))
		if err != nil {
			return nil, fmt.Errorf("failed to find disk image parts: %s", err)
		}
		totalParts := len(parts)
		for i, part := range parts {
			// Use 1-indexed part numbers.
			partNumber := i + 1
			files = append(files, fmt.Sprintf("%s:application/vnd.oci.image.layer.v1.tar;part.number=%d;part.total=%d", part, partNumber, totalParts))
		}
	}

	// Add config.json if available.
	if _, err := os.Stat(filepath.Join(workDir, "config.json")); err == nil {
		files = append(files, "config.json:application/vnd.oci.image.config.v1+json")
	}
	// Add nvram.bin if available.
	if _, err := os.Stat(filepath.Join(workDir, "nvram.bin")); err == nil {
		files = append(files, "nvram.bin:application/octet-stream")
	}
	return files, nil
}
