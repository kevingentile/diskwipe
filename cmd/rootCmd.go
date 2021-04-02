package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	stepSize int = 512 //bytes
)

var (
	targetDrive     string
	randomizeDelete bool
	deleteCycles    int
	shutdown        bool
)

func init() {
	rootCmd.Flags().StringVarP(&targetDrive, "drive", "d", "", "drive to wipe")
	rootCmd.Flags().BoolVarP(&randomizeDelete, "randomize", "r", false, "enable randomization of delete output")
	rootCmd.Flags().IntVarP(&deleteCycles, "cycles", "c", 3, "number of delete cycles")
	rootCmd.MarkFlagRequired("drive")
}

var rootCmd = &cobra.Command{
	Use:   "diskwipe",
	Short: "Wipe a disk provided a directory",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := parseFlags()
		if err != nil {
			panic(err)
		}

		r := newDeleteRunner(config)
		if err := r.run(); err != nil {
			panic(err)
		}

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func parseFlags() (*config, error) {
	info, err := os.Stat(targetDrive)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("invalid target drive: %s", targetDrive)
	}

	deviceSize := info.Size()
	if (info.Mode() & os.ModeDevice) != 0 {
		f, err := os.Open(targetDrive)
		deviceSize, err = f.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("failed to get device size")
		}
	} else {
		return nil, fmt.Errorf("device not found at: %s", targetDrive)
	}

	if deleteCycles <= 0 {
		return nil, fmt.Errorf("cycles must be greater than 1")
	}

	return &config{
		Target:     targetDrive,
		TargetInfo: info,
		TargetSize: deviceSize,
		Randomize:  randomizeDelete,
		Cycles:     deleteCycles,
		Shutdown:   shutdown,
	}, nil
}

type config struct {
	Target     string
	TargetInfo fs.FileInfo
	TargetSize int64
	Randomize  bool
	Cycles     int
	Shutdown   bool
}

type deleteRunner struct {
	Config *config
}

func newDeleteRunner(c *config) *deleteRunner {
	return &deleteRunner{Config: c}
}

func (d *deleteRunner) run() error {
	f, err := os.OpenFile(d.Config.Target, os.O_WRONLY, fs.ModeDevice)
	if err != nil {
		return err
	}

	offset := int64(0)
	buffer := make([]byte, stepSize)
	s := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(s)

	log.Println("Target Size:", d.Config.TargetSize)
	for offset < d.Config.TargetSize {
		if d.Config.Randomize {
			rng.Read(buffer)
		}
		n, err := f.WriteAt(buffer, int64(offset))
		if err != nil {
			return err
		}
		offset += int64(n)
	}

	return nil
}
