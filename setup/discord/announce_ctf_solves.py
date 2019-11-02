#!/usr/bin/env python3
#
# Polls the scoring engine, and announces the first solve of each challenge on Discord
#
#   Usage: ./announce_ctf_solves.py
#
import argparse
import datetime
import time
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
DISCORD_MESSAGE_TEMPLATE = '{team_name} was the first to solve "{challenge_name}" ({category}), scoring {points} points!'
# Template can include any of the following fields in this example:
#  { "category": "Trivia",
#    "challenge_id": 91,
#    "challenge_name": "Do you know your hacker history?",
#    "points": 25,
#    "team_id": 34,
#    "team_name": "team4",
#    "timestamp": "2019-09-21T09:13:42.712177-04:00" }



client = requests.Session()
client.cookies.update({'session': TOKEN})


def timestamp_now_in_rfc3339():
    now = datetime.datetime.now()
    return now.astimezone().isoformat(timespec='seconds').replace('+00:00', 'Z')

class Poller:
    poll_url = SERVER + "/api/public/ctf/solves"

    def __init__(self, interval=60):
        # `interval` is how often, in seconds, to check the scoring server for new solves
        self.interval = interval
        # `last_check_time` is the most recent timestamp of when the scoring server was polled
        self.last_check_time = timestamp_now_in_rfc3339()
        # `seen` holds all the challenge IDs that have already been solved before
        self.seen = set()

    def poll_for_solves(self):
        r = client.get(self.poll_url, params={'start_time': self.last_check_time}, verify=VERIFY_SSL)
        if r.status_code != requests.codes.OK:
            raise ValueError("Unexpected status code: {}; msg={}".format(r.status_code, r.text))
        return r.json()

    def filter_new_firsts(self, solves_json):
        new_firsts = {}

        for solve in solves_json:
            if solve["challenge_id"] not in self.seen:
                new_firsts[solve["challenge_id"]] = solve
        return new_firsts
    
    def step(self):
        try:
            solves_json = self.poll_for_solves()
        except ValueError as e:
            print("Error polling:", e)
            return

        self.last_check_time = timestamp_now_in_rfc3339()

        new_firsts = self.filter_new_firsts(solves_json)
        if len(new_firsts):
            for solve in new_firsts.values():
                try:
                    alert_on_discord(solve)
                except requests.HTTPError as e:
                    print("Error altering to discord:", e)
                    print("No alert was sent for {challenge_name} solve by {team_name}".format(**solve))
                    # keep going...

            self.seen.update(new_firsts.keys())
        
    def run(self):
        while True:
            self.step()
            time.sleep(self.interval)
            

def alert_on_discord(solve):
    # https://discordapp.com/developers/docs/resources/webhook#execute-webhook
    data = {
        'content': DISCORD_MESSAGE_TEMPLATE.format(**solve),
    }
    r = requests.post(DISCORD_WEBHOOK_URL, data=data, params={'wait': True})
    r.raise_for_status()

    # All set!
    print("Announcement sent:", data['content'])
    print("Discord response:", r.text)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument("--interval", type=int, default=60, 
        help="How often to poll the scoring server for new solves")
    ns = parser.parse_args()

    poller = Poller(interval=ns.interval)
    try:
        poller.run()
    except Exception as e:
        print("Fatal Error:", e)
        print(traceback.format_exc())
