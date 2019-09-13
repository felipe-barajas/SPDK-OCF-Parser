package main

import (
   "encoding/json"
    "fmt"
    "flag"
    "time"
    "os/exec"
    "strconv"
    "log"
    "os"

    "net/http"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
  portNumber int
  sleepTime int
  isLogEnabled bool
  logPath string
)

type Bdev struct {
  Name string
  Bytes_read float64
  Num_read_ops float64
  Bytes_written float64
  Num_write_ops float64
  Bytes_unmapped float64
  Num_unmap_os float64
  Read_latency_ticks float64
  Write_latency_ticks float64
  Unmap_latency_ticks float64
}

type TickRate struct {
  Tick_rate float64
}

type IOStat struct {
  Tick_rate float64
  Bdevs []Bdev
}

type OCF_data struct {
  Count  float64
  Percentage string
  Units string
}

type OCF_usage struct {
  Occupancy OCF_data
  Free OCF_data
  Clean OCF_data
  Dirty OCF_data
}

type OCF_requests struct {
  Rd_hits OCF_data
  Rd_partial_misses OCF_data
  Rd_full_misses OCF_data
  Rd_total OCF_data
  Wr_hits OCF_data
  Wr_partial_misses OCF_data
  Wr_full_misses OCF_data
  Wr_total OCF_data
  Rd_pt OCF_data
  Wr_pt OCF_data
  Serviced OCF_data
  Total OCF_data
}

type OCF_blocks struct {
  Core_volume_rd OCF_data
  Core_volume_wr OCF_data
  Core_volume_total OCF_data
  Cache_volume_rd OCF_data
  Cache_volume_wr OCF_data
  Cache_volume_total OCF_data
  Volume_rd OCF_data
  Volume_wr OCF_data
  Volume_total OCF_data
}

type OCF_errors struct {
  Core_volume_rd OCF_data
  Core_volume_wr OCF_data
  Core_volume_total OCF_data
  Cache_volume_rd OCF_data
  Cache_volume_wr OCF_data
  Cache_volume_total OCF_data
  Total OCF_data
}

type OCFStat struct {
  Usage OCF_usage
  Requests OCF_requests
  Blocks OCF_blocks
  Errors OCF_errors
}


var (
	IOStat_bytes_read = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_bytes_read",
			Help: "Number of bytes read",
		},
		[]string{"bdev_name"},
	)
  IOStat_read_ops = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_num_read_ops",
			Help: "Number of read operations",
		},
		[]string{"bdev_name"},
	)
  IOStat_bytes_written = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_bytes_written",
			Help: "Number of bytes written",
		},
		[]string{"bdev_name"},
	)
  IOStat_write_ops = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_num_write_ops",
			Help: "Number of write operations",
		},
		[]string{"bdev_name"},
	)
  IOStat_bytes_unmapped = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_bytes_unmapped",
			Help: "Number of bytes unmapped",
		},
		[]string{"bdev_name"},
	)
  IOStat_unmapped_ops = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_unmapped_ops",
			Help: "Number of unmapped ops",
		},
		[]string{"bdev_name"},
	)
  IOStat_read_latency_ticks = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_read_latency_ticks",
			Help: "Number of read latency ticks",
		},
		[]string{"bdev_name"},
	)
  IOStat_write_latency_ticks = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_write_latency_ticks",
			Help: "Number of write latency ticks",
		},
		[]string{"bdev_name"},
	)
  IOStat_unmap_latency_ticks = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_unmap_latency_ticks",
			Help: "Number of unmap latency ticks",
		},
		[]string{"bdev_name"},
	)
  IOStat_tick_rate = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "spdk_tick_rate",
			Help: "The tick rate",
	})

  OCFStat_count = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_ocf_count",
			Help: "OCF count value",
		},
		[]string{"category", "subcategory"},
  )
  OCFStat_percentage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdk_ocf_percentage",
			Help: "OCF percentage value",
		},
		[]string{"category", "subcategory"},
  )
)

func check(e error) {
  if e != nil {
    log.Fatal(e)
    panic(e)
  }
}

func recordMetrics() {
  go func() {
    for {
      var parsed_iostat_data IOStat

      io_stat_json_data,iostat_err := exec.Command("/root/spdk/scripts/rpc.py","get_bdevs_iostat").Output()

      xprint("SPDK IOSTAT DATA:\n" + fmt.Sprintln(parsed_iostat_data))
      if (iostat_err) != nil {
        continue
      }

      json.Unmarshal([]byte(io_stat_json_data), &parsed_iostat_data)

      if (len(parsed_iostat_data.Bdevs) < 1 ){
        //If unmarshal did not find the bdevs then SPDK is not providing the Bdevs with a key so get it seperately
        var tick_rates []TickRate
        json.Unmarshal([]byte(io_stat_json_data), &tick_rates)

        var bdevs []Bdev
        json.Unmarshal([]byte(io_stat_json_data), &bdevs)
        parsed_iostat_data.Bdevs = bdevs

        if (len(tick_rates) > 0){
          parsed_iostat_data.Tick_rate = tick_rates[0].Tick_rate
        }
      }

      var parsed_ocf_data OCFStat
      ocf_json_data,ocf_err := exec.Command("/root/spdk/scripts/rpc.py","get_ocf_stats", "Cache1").Output()

      xprint("SPDK OCF DATA:\n" + fmt.Sprint(string(ocf_json_data)))
      if (ocf_err) != nil {
        continue
      }

      json.Unmarshal([]byte(ocf_json_data), &parsed_ocf_data)

      IOStat_tick_rate.Add(parsed_iostat_data.Tick_rate)
      for _,bdev := range parsed_iostat_data.Bdevs {
        IOStat_bytes_read.With(prometheus.Labels{"bdev_name":bdev.Name}).Set(bdev.Bytes_read)
        IOStat_read_ops.With(prometheus.Labels{"bdev_name":bdev.Name}).Set( bdev.Num_read_ops)
        IOStat_bytes_written.With(prometheus.Labels{"bdev_name":bdev.Name}).Set( bdev.Bytes_written )
        IOStat_write_ops.With(prometheus.Labels{"bdev_name":bdev.Name}).Set( bdev.Num_write_ops)
        IOStat_bytes_unmapped.With(prometheus.Labels{"bdev_name":bdev.Name}).Set(bdev.Bytes_unmapped  )
        IOStat_unmapped_ops.With(prometheus.Labels{"bdev_name":bdev.Name}).Set(bdev.Num_unmap_os )
        IOStat_read_latency_ticks.With(prometheus.Labels{"bdev_name":bdev.Name}).Set(bdev.Read_latency_ticks )
        IOStat_write_latency_ticks.With(prometheus.Labels{"bdev_name":bdev.Name}).Set(bdev.Write_latency_ticks )
        IOStat_unmap_latency_ticks.With(prometheus.Labels{"bdev_name":bdev.Name}).Set(bdev.Unmap_latency_ticks )
      }

      OCFStat_count.With(prometheus.Labels{"category":"usage",    "subcategory":"occupancy"}).Set(parsed_ocf_data.Usage.Occupancy.Count)
      OCFStat_count.With(prometheus.Labels{"category":"usage",    "subcategory":"free"}).Set(parsed_ocf_data.Usage.Free.Count)
      OCFStat_count.With(prometheus.Labels{"category":"usage",    "subcategory":"clean"}).Set(parsed_ocf_data.Usage.Clean.Count)
      OCFStat_count.With(prometheus.Labels{"category":"usage",    "subcategory":"dirty"}).Set(parsed_ocf_data.Usage.Dirty.Count)

      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"rd_hits"}).Set(parsed_ocf_data.Requests.Rd_hits.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"rd_partial_misses"}).Set(parsed_ocf_data.Requests.Rd_partial_misses.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"rd_full_misses"}).Set(parsed_ocf_data.Requests.Rd_full_misses.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"rd_total"}).Set(parsed_ocf_data.Requests.Rd_total.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"rd_hits"}).Set(parsed_ocf_data.Requests.Rd_hits.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"wr_hits"}).Set(parsed_ocf_data.Requests.Wr_hits.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"wr_partial_misses"}).Set(parsed_ocf_data.Requests.Wr_partial_misses.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"wr_full_misses"}).Set(parsed_ocf_data.Requests.Wr_full_misses.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"wr_total"}).Set(parsed_ocf_data.Requests.Wr_total.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"rd_pt"}).Set(parsed_ocf_data.Requests.Rd_pt.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"wr_pt"}).Set(parsed_ocf_data.Requests.Wr_pt.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"serviced"}).Set(parsed_ocf_data.Requests.Serviced.Count)
      OCFStat_count.With(prometheus.Labels{"category":"requests", "subcategory":"total"}).Set(parsed_ocf_data.Requests.Total.Count)

      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"core_volume_rd"}).Set(parsed_ocf_data.Blocks.Core_volume_rd.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"core_volume_wr"}).Set(parsed_ocf_data.Blocks.Core_volume_wr.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"core_volume_total"}).Set(parsed_ocf_data.Blocks.Core_volume_total.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"cache_volume_rd"}).Set(parsed_ocf_data.Blocks.Cache_volume_rd.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"cache_volume_wr"}).Set(parsed_ocf_data.Blocks.Cache_volume_wr.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"cache_volume_total"}).Set(parsed_ocf_data.Blocks.Cache_volume_total.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"volume_rd"}).Set(parsed_ocf_data.Blocks.Volume_rd.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"volume_wr"}).Set(parsed_ocf_data.Blocks.Volume_wr.Count)
      OCFStat_count.With(prometheus.Labels{"category":"blocks",   "subcategory":"volume_total"}).Set(parsed_ocf_data.Blocks.Volume_total.Count)

      OCFStat_count.With(prometheus.Labels{"category":"errors",   "subcategory":"core_volume_rd"}).Set(parsed_ocf_data.Errors.Core_volume_rd.Count)
      OCFStat_count.With(prometheus.Labels{"category":"errors",   "subcategory":"core_volume_wr"}).Set(parsed_ocf_data.Errors.Core_volume_wr.Count)
      OCFStat_count.With(prometheus.Labels{"category":"errors",   "subcategory":"core_volume_total"}).Set(parsed_ocf_data.Errors.Core_volume_total.Count)
      OCFStat_count.With(prometheus.Labels{"category":"errors",   "subcategory":"cache_volume_rd"}).Set(parsed_ocf_data.Errors.Cache_volume_rd.Count)
      OCFStat_count.With(prometheus.Labels{"category":"errors",   "subcategory":"cache_volume_wr"}).Set(parsed_ocf_data.Errors.Cache_volume_wr.Count)
      OCFStat_count.With(prometheus.Labels{"category":"errors",   "subcategory":"cache_volume_total"}).Set(parsed_ocf_data.Errors.Cache_volume_total.Count)
      OCFStat_count.With(prometheus.Labels{"category":"errors",   "subcategory":"total"}).Set(parsed_ocf_data.Errors.Total.Count)


      if s,err := strconv.ParseFloat(parsed_ocf_data.Usage.Occupancy.Percentage             ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"usage",    "subcategory":"occupancy"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Usage.Free.Percentage                  ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"usage",    "subcategory":"free"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Usage.Clean.Percentage                 ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"usage",    "subcategory":"clean"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Usage.Dirty.Percentage                 ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"usage",    "subcategory":"dirty"}).Set(s)}

      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Rd_hits.Percentage            ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"rd_hits"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Rd_partial_misses.Percentage  ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"rd_partial_misses"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Rd_full_misses.Percentage     ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"rd_full_misses"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Rd_total.Percentage           ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"rd_total"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Rd_hits.Percentage            ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"rd_hits"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Wr_hits.Percentage            ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"wr_hits"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Wr_partial_misses.Percentage  ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"wr_partial_misses"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Wr_full_misses.Percentage     ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"wr_full_misses"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Wr_total.Percentage           ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"wr_total"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Rd_pt.Percentage              ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"rd_pt"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Wr_pt.Percentage              ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"wr_pt"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Serviced.Percentage           ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"serviced"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Requests.Total.Percentage              ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"requests", "subcategory":"total"}).Set(s)}

      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Core_volume_rd.Percentage       ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"core_volume_rd"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Core_volume_wr.Percentage       ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"core_volume_wr"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Core_volume_total.Percentage    ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"core_volume_total"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Cache_volume_rd.Percentage      ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"cache_volume_rd"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Cache_volume_wr.Percentage      ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"cache_volume_wr"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Cache_volume_total.Percentage   ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"cache_volume_total"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Volume_rd.Percentage            ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"volume_rd"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Volume_wr.Percentage            ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"volume_wr"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Blocks.Volume_total.Percentage         ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"blocks",   "subcategory":"volume_total"}).Set(s)}

      if s,err := strconv.ParseFloat(parsed_ocf_data.Errors.Core_volume_rd.Percentage       ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"errors",   "subcategory":"core_volume_rd"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Errors.Core_volume_wr.Percentage       ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"errors",   "subcategory":"core_volume_wr"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Errors.Core_volume_total.Percentage    ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"errors",   "subcategory":"core_volume_total"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Errors.Cache_volume_rd.Percentage      ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"errors",   "subcategory":"cache_volume_rd"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Errors.Cache_volume_wr.Percentage      ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"errors",   "subcategory":"cache_volume_wr"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Errors.Cache_volume_total.Percentage   ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"errors",   "subcategory":"cache_volume_total"}).Set(s)}
      if s,err := strconv.ParseFloat(parsed_ocf_data.Errors.Total.Percentage                ,64); err == nil { OCFStat_percentage.With(prometheus.Labels{"category":"errors",   "subcategory":"total"}).Set(s)}

      time.Sleep(time.Duration(sleepTime) * time.Second)
    }
  }()
}

func init() {
  prometheus.MustRegister(IOStat_bytes_read)
  prometheus.MustRegister(IOStat_read_ops)
  prometheus.MustRegister(IOStat_bytes_written)
  prometheus.MustRegister(IOStat_write_ops)
  prometheus.MustRegister(IOStat_bytes_unmapped)
  prometheus.MustRegister(IOStat_unmapped_ops)
  prometheus.MustRegister(IOStat_read_latency_ticks)
  prometheus.MustRegister(IOStat_write_latency_ticks)
  prometheus.MustRegister(IOStat_unmap_latency_ticks)
  prometheus.MustRegister(IOStat_tick_rate)
  prometheus.MustRegister(OCFStat_count)
  prometheus.MustRegister(OCFStat_percentage)
}

func xprint( message string){
  // Max log file size in bytes
  var maxFileSize int64 = 104857600
  var didTrim bool = false

  if (isLogEnabled == false){
    return
  }

   fileStat, err := os.Stat(logPath)
   if err == nil  {
     if fileStat.Size() > maxFileSize {
       err = os.Rename(logPath, logPath + ".old")
       if err != nil {
         fmt.Println("Failed to rename file")
         fmt.Println(err)
       }else {
         didTrim = true
       }
     }
   } else {
     fmt.Println("Failed to list file")
     fmt.Println(err)
   }

    currentTime := time.Now()
    formattedMsg := fmt.Sprintln(message)
    line := "MSG : " + currentTime.Format("01/02/2006 3:4:5 PM") + " - " + formattedMsg

    f, file_err := os.OpenFile(logPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

  	if file_err != nil {
      fmt.Println("Failed to write to log file")
  		fmt.Println(file_err)
      return
  	}
  	defer f.Close()
    if didTrim {
      fmt.Fprintf(f, "Old Files have been moved to " + logPath + ".old \n")
    }
  	fmt.Fprintf(f, "%s\n", line)
}

func main() {

  //argument functions, default values, help text
  portPtr := flag.Int("port", 2113, "The port number to provide metrics to")
  sleepPtr := flag.Int("sleep", 1, "The number of seconds to sleep in between metrics")
  logPtr := flag.Bool("log", false, "Turns on logging information")
  logPathPtr := flag.String("logfile", "/tmp/spdk_parser.out", "log file location")

  flag.Parse()

  portNumber = *portPtr
  sleepTime = *sleepPtr
  isLogEnabled = *logPtr
  logPath = *logPathPtr

  port := ":" + strconv.Itoa(portNumber)

  xprint("### Starting Execution of spdk_parser...")
  xprint("Port         :" + strconv.Itoa(portNumber))
  xprint("Sleep Time   :" + strconv.Itoa(sleepTime))
  xprint("isLogEnabled :" + strconv.FormatBool(isLogEnabled))
  xprint("Log Path     :" + logPath)
  xprint("Other Args   :" + fmt.Sprintln(flag.Args()))

  recordMetrics()

  http.Handle("/metrics", promhttp.Handler())

  log.Fatal(http.ListenAndServe(port, nil))
}
