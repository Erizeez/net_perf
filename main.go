package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"regexp"
	"strconv"
	"strings"

	"github.com/jony-lee/go-progress-bar"
	"github.com/spf13/pflag"
)

// iperf stat struct
type IperfStat struct {
	Interval      string
	Transfer      float64
	TransferUnit  string
	Bandwidth     float64
	BandwidthUnit string
	Write         int64
	Err           int64
	Rtry          int64
	Cwnd          string
	RTT           string
	NetPwr        int64
}

// Run iperf and collect stats

func runIperf(iperf_bin string, ip string, iperf_time int, iperf_interval int) []IperfStat {
	// Start a progress bar
	progress_total := int(iperf_time) * 10
	bar := progress.New(int64(progress_total))
	// Start a goroutine to update the progress bar
	go func() {
		for i := 0; i < progress_total; i++ {
			time.Sleep(time.Second / 10)
			bar.Done(1)
		}
	}()

	// Run iperf
	cmd := exec.Command(iperf_bin, "-c", ip, "-t", fmt.Sprintf("%d", iperf_time), "-i", fmt.Sprintf("%d", iperf_interval), "-e")
	output, err := cmd.Output()

	if err != nil {
		fmt.Println("Iperf exec error", err)
		os.Exit(1)
	}

	// Collect stats
	fmt.Println(string(output))
	fmt.Println("Iperf exec success")

	// Extract stats from output by each line using regex
	re := regexp.MustCompile(`^\[\s \d\]\s.*\ssec.*`)
	stats := make([]IperfStat, 0)
	for _, line := range strings.Split(string(output), "\n") {
		if re.MatchString(line) {
			fields := strings.Fields(line)

			transfer_stat, err := strconv.ParseFloat(fields[4], 64)
			if err != nil {
				fmt.Println("Stat extract error", err)
				os.Exit(1)
			}

			bandwidth_stat, err := strconv.ParseFloat(fields[6], 64)
			if err != nil {
				fmt.Println("Stat extract error", err)
				os.Exit(1)
			}

			write_err := strings.Split(fields[8], "/")

			write_stat, err := strconv.ParseInt(write_err[0], 10, 64)
			if err != nil {
				fmt.Println("Stat extract error", err)
				os.Exit(1)
			}
			err_stat, err := strconv.ParseInt(write_err[0], 10, 64)
			if err != nil {
				fmt.Println("Stat extract error", err)
				os.Exit(1)
			}
			rtry_stat, err := strconv.ParseInt(fields[9], 10, 64)
			if err != nil {
				fmt.Println("Stat extract error", err)
				os.Exit(1)
			}

			netpwr_stat, err := strconv.ParseInt(fields[12], 10, 64)
			if err != nil {
				fmt.Println("Stat extract error", err)
				os.Exit(1)
			}

			stats = append(stats, IperfStat{
				Interval:      fields[2],
				Transfer:      transfer_stat,
				TransferUnit:  fields[5],
				Bandwidth:     bandwidth_stat,
				BandwidthUnit: fields[7],
				Write:         write_stat,
				Err:           err_stat,
				Rtry:          rtry_stat,
				Cwnd:          strings.Split(fields[10], "/")[0],
				RTT:           strings.Split(fields[10], "/")[1] + " " + fields[11],
				NetPwr:        netpwr_stat,
			})
		}
	}
	bar.Finish()

	// Return stats
	return stats
}

func main() {
	// Generate default output filename by timestamp
	t := time.Now()
	default_output := fmt.Sprintf("iperf_%d%02d%02d_%02d%02d%02d.csv", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

	// Parse Args
	iperf_time := pflag.IntP("iperf_time", "t", 30, "time to wait")
	iperf_interval := pflag.IntP("iperf_interval", "i", 1, "interval to wait")
	iperf_bin := pflag.String("iperf_bin", "iperf", "iperf binary path")
	ip := pflag.StringP("ip", "c", "0.0.0.0", "ip to connect")
	output := pflag.StringP("output", "o", default_output, "output file")
	help := pflag.BoolP("help", "h", false, "show help")

	pflag.Parse()

	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// Show Config
	fmt.Println("Iperf_time:", *iperf_time)
	fmt.Println("IP:", *ip)

	// Run Iperf
	iperf_stats := runIperf(*iperf_bin, *ip, *iperf_time, *iperf_interval)
	fmt.Printf("%+v\n", iperf_stats)

	// Save stats to csv
	f, err := os.Create(*output)
	if err != nil {
		fmt.Println("File create error", err)
		os.Exit(1)
	}
	defer f.Close()

	// Write header
	f.WriteString("Interval,Transfer,TransferUnit,Bandwidth,BandwidthUnit,Write,Err,Rtry,Cwnd,RTT,NetPwr\n")

	// Write stats
	for _, stat := range iperf_stats {
		f.WriteString(fmt.Sprintf("%s,%f,%s,%f,%s,%d,%d,%d,%s,%s,%d\n", stat.Interval, stat.Transfer, stat.TransferUnit, stat.Bandwidth, stat.BandwidthUnit, stat.Write, stat.Err, stat.Rtry, stat.Cwnd, stat.RTT, stat.NetPwr))
	}

	// Show stats
	fmt.Println("Iperf stats saved to iperf.csv")

}
