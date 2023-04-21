# cf-ddns: Setup DDNS using Cloudflare API

cf-ddns is a CLI application developed purely in Golang, which enables you to set up a Dynamic DNS (DDNS) like feature on your host machine using the Cloudflare API. This application is a port of the [cloudflare-ddns-updater project](https://github.com/K0p1-Git/cloudflare-ddns-updater), originally developed as a shell script.

## Prerequisites

    A Domain name you own and is configured to use Cloudflare DNS.
    Cloudflare API key and zone ID
    An existing subdomain hosted on Cloudflare
    Golang installed on the host machine

## Setup
- In your Cloudflare DNS settings page for your domain name, create a subdomain with a random A record (I recommend 8.8.8.8) for ease. 
- From your Cloudflare dashboard, copy the Zone Identifier.
- Next, you need the authentication key. You may use the Global Key (which I don't recommend since it gives complete access to your account) or create a Scoped Token with limited permissions.
- Clone this repo:

```bash
git clone https://github.com/ninostephen/cf-ddns.git
```

- Install the required packages:
```bash
cd cf-ddns
go mod tidy
```
- Now that you have all the required golang modules, you can update the app.env file. 
```
# Use email address used to login to 'https://dash.cloudflare.com'
AUTH_EMAIL=

# Set to global for Global API Key or token for Scoped API Token. Use Scoped API in most cases.
AUTH_METHOD=global

# Your API Token or Global API Key. You can find it in your cloudflare dashboard
AUTH_KEY=

# Can be found in the Overview tab of your domain
ZONE_IDENTIFIER=

# Which record you want to be synced. This is the subdomain name you want to update
RECORD_NAME=

# Set the DNS TTL (seconds)
TTL=3600

# Set the proxy to true or false
PROXY=false

# Title of site "Example Site"
SITENAME=

# Set this to true if you want a more verbose output. 
VERBOSE=false
```
- Once you have filled in all the above env variables, build the application:

```bash
go build
```
- You can try running it to make sure its working. 
```
./cf-ddns
```
- The next step is to setup a cronjob.
```bash
ctrontab -e
```

You can add the following line in your cron job file. Make sure to update the path to binary.
```
*/1 * * * *  /bin/bash /path/to/binary/cf-ddns
```
This cron job will run every 1 minute to check your public IP address. The cf-ddns application will only update the A record when it sees the IP address has changed.

## Setup Port Forwarding
To reap the benefits of using this application, you have to open up ports on your router and forward them to your local machine. Setting up port forwarding is a topic outside the scope of documentation. You may find the required documentation on Google for your specific router. 

## License

cf-ddns is licensed under the MIT License. See the LICENSE file for more details.

## Acknowledgments

This project is a port of the cloudflare-ddns-updater project by K0p1-Git. Special thanks to K0p1-Git for developing the original project.
