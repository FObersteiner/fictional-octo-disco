# // Syntax
# <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]

# // Example
# myMeasurement,tag1=value1,tag2=value2 fieldKey="fieldValue" 1556813561098000000

# Wohnzimmer,id=58 T=13.544 1672099442
# Wohnzimmer,id=58 rH=55.04 1672099442

# import time
# from datetime import datetime, timezone
from pathlib import Path

from influxdb_client import InfluxDBClient, Point  # , WritePrecision
from influxdb_client.client.write_api import SYNCHRONOUS
import pandas as pd

#token = "dVAmkCxXIgjCPqi9ckqZbgn0YqKUCPdCMaSXEyl0Mj5E39mEmjIaCBVVAclaUT7O2XtYhV1_t9KhTbHb50yxOQ=="
token = "t7tlm4e_uZYV_AldiIwh71LUQXhR_3RmFwNPTYHRuh2gOH3yTsLkX8Clht0B2xC7RZWaUD2mxCt3uuhK_dsT-w=="
org = "7645684d91d8ce5f"
url = "192.168.0.108:8086"
bucket = "solltIchLueften"
client = InfluxDBClient(url=url, token=token, org=org)
write_api = client.write_api(write_options=SYNCHRONOUS)

# file = Path("./dump_20230103/DATA/sensorlogger/20221229Z_sensordata.csv").resolve()
# df = pd.read_csv(file, sep=";")

files = sorted(Path("./dump_20230103/DATA/sensorlogger/").glob("*_sensordata.csv"))
for idx, f in enumerate(files):
    df = pd.read_csv(f, sep="\t")
    print(f"{idx+1} of {len(files)}: {f.name}")
    for row in df.iterrows():
        # print(f"  ... row {row[0]+1} of {df.shape[0]}")
        fields = row[1]
        records = [
            Point(fields["name"])
            .tag("id", fields["id"])
            .field("T", fields["temp_degC"])
            .time(fields["datetime"].replace("+00:00", "Z"))
        ]
        records += [
            Point(fields["name"])
            .tag("id", fields["id"])
            .field("rH", fields["relHum_%"])
            .time(fields["datetime"].replace("+00:00", "Z"))
        ]
        records += [
            Point(fields["name"])
            .tag("id", fields["id"])
            .field("aH", fields["absHum_gkg"])
            .time(fields["datetime"].replace("+00:00", "Z"))
        ]
        if fields["pres_hPa"] > 100:
            records += [
                Point(fields["name"])
                .tag("id", fields["id"])
                .field("p", fields["pres_hPa"])
                .time(fields["datetime"].replace("+00:00", "Z"))
            ]
            write_api.write(bucket=bucket, org=org, record=records)
    print(f"    done uploadeing {f.name}")

print("all done.")
