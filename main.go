package main

import (
  "encoding/json"
  "fmt"
  "net/http"
  "strconv"
  "os"
  "io/ioutil"
  "io"
  "sync"
  "flag"
)

type Config struct {
  channel string
  folder string
  per int
  page int
  length int
  // checked int
  // downloaded int
}

type Arena struct {
  Length int
  Title string
  Created_At string
  Contents []Content
}

type Content struct {
  Id int
  Title string
  Class string
  Image Image
}

type Image struct {
  Filename string
  Content_Type string
  Original struct {
    Url string
  }
}

func makeRequest (c *Config) {
  fmt.Println("making request...")
  c.page++

  // setup sync group for this request func
  var wg sync.WaitGroup

  per := strconv.Itoa(c.per)
  page := strconv.Itoa(c.page)

  url := fmt.Sprintf("https://api.are.na/v2/channels/%s?per=%s&direction=desc&page=%s", c.channel, per, page)
  resp, err := http.Get(url)

  if (err != nil) {
    fmt.Println(err)
  }

  defer resp.Body.Close()

  body, readErr := ioutil.ReadAll(resp.Body)
  if (readErr != nil) {
    fmt.Println(readErr)
  }

  var b Arena
  jsonErr := json.Unmarshal([]byte(body), &b)

  if (jsonErr != nil) {
    fmt.Println(jsonErr)
  }

  c.length = b.Length

  reqLen := len(b.Contents)
  fmt.Println("attempting to download", reqLen)

  // iterate over images
  for _, image := range b.Contents {
    if (image.Class == "Image") {
        // add to page group for concurrency
        wg.Add(1)
	go func(image *Image) {
	  downloadImage(image.Original.Url, image.Filename, c)
	  defer wg.Done()
        }(&image.Image)  
    }
    // wait for sync group to iterate over all element
    wg.Wait()
  }

  if (reqLen == c.per) {
    fmt.Println("NEXT SET")
    // keep waiting
    wg.Wait()
    makeRequest(c) 
  } else {
    fmt.Println("END")
  }
}

func downloadImage (url string, filename string, c *Config) {
  res, err := http.Get(url)
  if err != nil {
    fmt.Println(err)
  }

  defer res.Body.Close()

  fileUrl := fmt.Sprintf("./%s/%s.jpg", c.folder, filename)

  file, err := os.Create(fileUrl)
  if err != nil {
    fmt.Println(err)
  }
  defer file.Close()

  // copy contents to file
  _, err = io.Copy(file, res.Body)
  if err != nil {
    fmt.Println(err)
  }
  fmt.Println(filename, "success")
}

func setupConfig (c *Config) {
  // channel = red-n-black
  // folder = images

  channel := flag.String("channel", "red-n-black", "are.na channel")
  folder := flag.String("folder", "images", "folder for images")
  flag.Parse()

  c.channel = *channel
  c.folder = *folder

  folderUrl := string(*folder)
  if _, err := os.Stat(folderUrl); os.IsNotExist(err) {
   os.MkdirAll(folderUrl, os.ModePerm)
  }

  fmt.Println(*channel, *folder)
}

func main () {
  config := Config{
    channel: "type-ideas",
    per: 64,
    page: 0,
  }

  setupConfig(&config)
  makeRequest(&config)
}

