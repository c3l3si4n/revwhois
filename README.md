# revwhois
CLI tool for discovering related base domains using WhoisXMLAPI's reverse Whois endpoints


## Usage

```bash
$ revwhois -h
Usage of ./revwhois:
  -domain string
    	Input domain
  -mode string
    	Mode (purchase, or preview) (default "purchase")
  -output string
    	Output file
  -searchtype string
    	Search type (current, or historic) (default "current")
  -token string
    	API token
```

You can preview the results of the reverse whois query by using the `preview` mode. This will return the number of results that would be returned if you were to purchase the results.

```bash
revwhois -domain example.com -mode preview
```

You can purchase the results of the reverse whois query by using the `purchase` mode. This will return the results of the reverse whois query and write them to a file.

```bash
revwhois -domain example.com -mode purchase -output example_com_domains.txt
```



## Installation

```bash
go install github.com/c3l3si4n/revwhois@HEAD
```
