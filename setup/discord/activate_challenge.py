#!/usr/bin/env python3
#
# Pass in a challenge ID to activate it and then announce it on Discord.
#   Usage: ./activate_challenge 102
#
import argparse
import traceback

import requests

### CONFIG ###

# TOKEN is your session cookie. Copy it from the browser, it should look similar to this (which is no longer valid):
TOKEN="wbOBmP2cRHI9p98mlEVMNKPBplMZdln8rGM5-HI5ziPOTq3Le-eLq_XDpsBdxPY4FF81oHkNymFHCKDfdc0gHp_D4BOcYd2EODCYIklDzxyBUkHXG3dzx5NcP1wNCSQCZySAjwcJCfPuoERP"
# SERVER is where the scoring engine is hosted.
SERVER="https://score.cnyhackathon.org:8081"
# If the server uses untrusted certs, you'll need to toggle this to false
VERIFY_SSL=True
# Discord webhook (read more: https://support.discordapp.com/hc/en-us/articles/228383668).
# Make sure you set this up with the right User & Room!
DISCORD_WEBHOOK_URL = "https://discordapp.com/api/webhooks/<...webhook_url...>"
# Message sent over discord.
DISCORD_MESSAGE_TEMPLATE = '@Blue Team : We just released "{name}" in the {category} category, worth "{total}" points!'
# Template can include any of the following fields in this example:
#   { "id":102,
#     "name":"Find the flag", 
#     "category":"Web",
#     "designer":"Mike",
#     "total":50,
#     "body":"...<description>...",
#     "flag":"...<secret>...",
#     "<some more things>": "..."}



client = requests.Session()
client.cookies.update({'session': TOKEN})

def get_challenge_info(flag_id):
    url = SERVER + "/api/ctf/flags/{id}".format(id=flag_id)
    r = client.get(url, verify=VERIFY_SSL)
    r.raise_for_status()
    return r.json()

def activate_challenge(flag_id):
    url = SERVER + "/api/ctf/flags/{id}/activate".format(id=flag_id)
    r = client.post(url, verify=VERIFY_SSL)
    r.raise_for_status()
    print("Challenge '{}' activated.".format(flag_id))

def alert_on_discord(chal):
    # https://discordapp.com/developers/docs/resources/webhook#execute-webhook
    data = {
        'content': DISCORD_MESSAGE_TEMPLATE.format(**chal),
    }
    r = requests.post(DISCORD_WEBHOOK_URL, data=data, params={'wait': True})
    r.raise_for_status()

    # All set!
    print("Discord notified:", r.text)

def main(flag_id):
    chal = get_challenge_info(flag_id)
    if chal["hidden"] == False:
        print("Challenge '{}' is already unlocked.".format(flag_id))
        return
    activate_challenge(flag_id)
    alert_on_discord(chal)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument("flag_id", type=int)
    ns = parser.parse_args()

    try:
        main(ns.flag_id)
    except Exception as e:
        print("Error:", e)
        print(traceback.format_exc())
