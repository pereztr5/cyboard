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
