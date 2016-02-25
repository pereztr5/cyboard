# Cyboard

This is a scoring engine for cyber defense competitions. This currenlty includes:  
- Database (MongoDB)  
- Front-End  
- CTF API  
- Service Checker  
- Team Score Web Socket  

### Configuration
You can use a configuration file to set the setting for the program. The file
name is config.toml. There are 2 different location you can have the config
file, in order (last location found will be used):  
- $HOME/.cyboard/config.toml  
- . (current director)

There are multiple ways to configure the MongoDB URI(Last location found will be used):
- In the config file like so:  
    `[database]
        uri = "mongodb://127.0.0.1"`  
- As a command line parameter  
    - `--mongodb_uri "mongodb://127.0.0.1"`  
- As a environment variable  
    - `MONGODB_URI`  

Flags will overwrite setting in the configuration file

### Installation

To get **cyboard** up and running with some testing data:

1. Ensure **mongodb** is installed and active
2. Configure **_config.toml_** with the correct settings for your installation of mongodb
3. Import test data:
    - `sh ./dbcommands.sh`
4. Build cyboard:
    - `go build`
5. Start the server:
    - `./cyboard server`
6. Take it for a spin:
    - ex: `http://127.0.0.1:8080`

_Note on SSL:_ To quickly enable ssl using self-signed certs, you may run:
    `go run ./certs/generate_cert.go --host https://127.0.0.1:5433 --rsa-bits 4048`
