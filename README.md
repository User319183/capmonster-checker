# CapMonster Checker

This project is a Go application that generates and checks keys for CapMonster.

## Features

- Generates a specified number of keys
- Checks the validity of the keys
- Uses multiple workers for efficient key checking
- Logs valid and invalid keys

## Prerequisites

- Go 1.x
- A proxy server

## Installation

Clone the repository:

```sh
git clone https://github.com/user319183/capmonster-checker.git
```

## Usage
To run the application, use the following command:
    
```sh
go run main.go
```
You can specify the number of workers, the file with keys, the API endpoint, and the proxy details as command-line flags.

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
MIT