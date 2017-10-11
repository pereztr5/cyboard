# Cyboard

[![Build Status](https://travis-ci.org/pereztr5/cyboard.svg?branch=master)](https://travis-ci.org/pereztr5/cyboard)

This is a scoring engine for cyber defense competitions. This currently includes:
- Database (MongoDB)  
- Front-End  
- CTF API  
- Service Checker  
- Team Score Web Socket  

### Background
I am doing this project as my capstone, senior project, at SUNY Polytechnic and to use
for the [CNY Hackathon](https://www.cnyhackathon.org "CNY Hackathon Home"). The goal
is to make a modular Scoring Engine. The approach for the service checking is currently
to accept any script and base the it on the exit status code. There are different
things currently implemented as listed above but the goal is to make is modular, for
example you could only use the service checking portion which provides an API to use
and use our own web appication. 

This is a work in progress and all suggestions are welcome.  
Here are links to contacting me E-Mail: [t@ynotperez.net](mailto:t@ynotperez.net), Twitter: [@tonyxpr5](https://twitter.com/tonyxpr5) 

### Configuration
You can use a configuration file to set the setting for the program. The file
name is config.toml. There are 2 different location you can have the config
file, in order (last location found will be used):  
- $HOME/.cyboard/config.toml  
- . (current directory)

There are multiple ways to configure the MongoDB URI(Last location found will be used):
- In the config file like so:  
```
[database]
uri = "mongodb://127.0.0.1"
```
- As a command line parameter  
    - `--mongodb_uri "mongodb://127.0.0.1"`  
- As a environment variable  
    - `MONGODB_URI`  

Flags will overwrite setting in the configuration file

### Installation

To get **cyboard** up and running with some testing data:

1. Ensure **mongodb** is installed and active
2. Configure **_config.toml_** with the correct settings for your installation of mongodb
3. There are example data modals in the **setup/mongo** directory
4. Import test data:  
    - `cd setup`  
    - `sh ./dbcommands.sh`
5. Build cyboard:
    - `go build`
6. Start the server:
    - `./cyboard server`
7. Take it for a spin:
    - ex: `http://127.0.0.1:8080`

_Note on SSL:_ To quickly enable ssl using self-signed certs, you may run:  
    `go run ./setup/generate_cert.go --host https://127.0.0.1:5433 --rsa-bits 4048`

[LICENSE](LICENSE.txt)
