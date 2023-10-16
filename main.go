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
	progress_total := iperf_time * 10
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
	// fmt.Println(string(output))
	// fmt.Println("Iperf exec success")

	// Extract stats from output by each line using regex
	re := regexp.MustCompile(`^\[\s \d\]\s.*\ssec.*`)
	stats := make([]IperfStat, 0)
	for _, line := range strings.Split(string(output), "\n") {
		if re.MatchString(line) {
			fields := strings.Fields(line)

			transfer_stat, err := strconv.ParseFloat(fields[4], 64)
			if err != nil {
				fmt.Println("Iperf stat extract error", err)
				os.Exit(1)
			}

			bandwidth_stat, err := strconv.ParseFloat(fields[6], 64)
			if err != nil {
				fmt.Println("Iperf stat extract error", err)
				os.Exit(1)
			}

			write_err := strings.Split(fields[8], "/")

			write_stat, err := strconv.ParseInt(write_err[0], 10, 64)
			if err != nil {
				fmt.Println("Iperf stat extract error", err)
				os.Exit(1)
			}
			err_stat, err := strconv.ParseInt(write_err[0], 10, 64)
			if err != nil {
				fmt.Println("Iperf stat extract error", err)
				os.Exit(1)
			}
			rtry_stat, err := strconv.ParseInt(fields[9], 10, 64)
			if err != nil {
				fmt.Println("Iperf stat extract error", err)
				os.Exit(1)
			}

			netpwr_stat, err := strconv.ParseInt(fields[12], 10, 64)
			if err != nil {
				fmt.Println("Iperf stat extract error", err)
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

type PingStat struct {
	Entry    []PingStatEntry
	Transmit int64
	Receive  int64
	Loss     float64
	Min      float64
	Avg      float64
	Max      float64
	Stddev   float64
}

type PingStatEntry struct {
	ICMPSeq int64
	TTL     int64
	Time    float64
}

func runPing(ping_bin string, ip string, ping_time int) PingStat {
	// Start a progress bar
	progress_total := ping_time * 10
	bar := progress.New(int64(progress_total))
	go func() {
		for i := 0; i < progress_total; i++ {
			time.Sleep(time.Second / 10)
			bar.Done(1)
		}
	}()

	cmd := exec.Command(ping_bin, "-c", fmt.Sprintf("%d", ping_time), ip)
	output, err := cmd.Output()

	if err != nil {
		fmt.Println("Ping exec error", err)
		os.Exit(1)
	}

	re := regexp.MustCompile(`icmp_seq=(\d+) ttl=(\d+) time=([\d.]+) ms`)
	stat_entry := make([]PingStatEntry, 0)

	for _, line := range strings.Split(string(output), "\n") {
		// fmt.Println(line)
		match := re.FindStringSubmatch(line)
		if len(match) > 0 {
			icmp_stat, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				fmt.Println("Ping icmp_stat extract error", err)
				os.Exit(1)
			}
			ttl_stat, err := strconv.ParseInt(match[2], 10, 64)
			if err != nil {
				fmt.Println("Ping ttl_stat extract error", err)
				os.Exit(1)
			}
			time_stat, err := strconv.ParseFloat(match[3], 64)
			if err != nil {
				fmt.Println("Ping time_stat extract error", err)
				os.Exit(1)
			}

			stat_entry = append(stat_entry, PingStatEntry{
				ICMPSeq: icmp_stat,
				TTL:     ttl_stat,
				Time:    time_stat})

		}
	}
	stats := PingStat{
		Entry: stat_entry,
	}

	stat_re := regexp.MustCompile(`(\d+) packets transmitted, (\d+) received, (\d+)% packet loss, time (\d+)ms\nrtt min/avg/max/mdev = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)

	match := stat_re.FindStringSubmatch(string(output))
	if len(match) > 0 {
		transmitted := match[1]
		received := match[2]
		loss := match[3]
		min := match[5]
		avg := match[6]
		max := match[7]
		stddev := match[8]

		stats.Transmit, err = strconv.ParseInt(transmitted, 10, 64)
		if err != nil {
			fmt.Println("Ping stat transmit extract error", err)
			os.Exit(1)
		}
		stats.Receive, err = strconv.ParseInt(received, 10, 64)
		if err != nil {
			fmt.Println("Ping stat receive extract error", err)
			os.Exit(1)
		}
		stats.Loss, err = strconv.ParseFloat(loss, 64)
		if err != nil {
			fmt.Println("Ping stat loss extract error", err)
			os.Exit(1)
		}
		stats.Min, err = strconv.ParseFloat(min, 64)
		if err != nil {
			fmt.Println("Ping stat min extract error", err)
			os.Exit(1)
		}
		stats.Avg, err = strconv.ParseFloat(avg, 64)
		if err != nil {
			fmt.Println("Ping stat avg extract error", err)
			os.Exit(1)
		}
		stats.Max, err = strconv.ParseFloat(max, 64)
		if err != nil {
			fmt.Println("Ping stat max extract error", err)
			os.Exit(1)
		}
		stats.Stddev, err = strconv.ParseFloat(stddev, 64)
		if err != nil {
			fmt.Println("Ping stat stddev extract error", err)
			os.Exit(1)
		}
	}

	bar.Done(10)
	bar.Finish()

	return stats
}

func main() {
	// Generate default output filename by timestamp
	t := time.Now()
	default_output := fmt.Sprintf("iperf_%d%02d%02d_%02d%02d%02d.csv", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

	// Parse Args
	time := pflag.IntP("time", "t", 30, "time to perf")
	iperf_interval := pflag.IntP("iperf_interval", "i", 1, "interval to wait")
	iperf_bin := pflag.String("iperf_bin", "iperf", "iperf binary path")
	ping_bin := pflag.String("ping_bin", "ping", "ping binary path")
	ip := pflag.StringP("ip", "c", "127.0.0.1", "ip to connect")
	output := pflag.StringP("output", "o", default_output, "output file")
	help := pflag.BoolP("help", "h", false, "show help")

	pflag.Parse()

	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// Show Config
	fmt.Println("Perf_time:", *time)
	fmt.Println("Iperf_interval:", *iperf_interval)
	fmt.Println("IP:", *ip)
	fmt.Println("Iperf_bin:", *iperf_bin)
	fmt.Println("Ping_bin:", *ping_bin)
	fmt.Printf("Output: ./%s\n", *output)

	// Run Iperf
	iperf_stats := runIperf(*iperf_bin, *ip, *time, *iperf_interval)
	// fmt.Printf("%+v\n", iperf_stats)
	ping_stats := runPing(*ping_bin, *ip, *time)
	// fmt.Printf("%+v\n", ping_stats)

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
	f.WriteString("\n")

	// Show stats
	fmt.Println("Iperf stats saved to iperf.csv")

	// Save ping_stats to csv and append to above
	f.WriteString("Transmit,Receive,Loss,Min,Avg,Max,Stddev\n")
	f.WriteString(fmt.Sprintf("%d,%d,%f,%f,%f,%f,%f\n", ping_stats.Transmit, ping_stats.Receive, ping_stats.Loss, ping_stats.Min, ping_stats.Avg, ping_stats.Max, ping_stats.Stddev))
	f.WriteString("\n")

	f.WriteString("ICMPSeq,TTL,Time\n")
	for _, stat := range ping_stats.Entry {
		f.WriteString(fmt.Sprintf("%d,%d,%f\n", stat.ICMPSeq, stat.TTL, stat.Time))
	}

	// Show stats
	fmt.Println("Ping stats saved to ping.csv")

}
