from pathlib import Path
import json

import pandas as pd


# --------- older format to new -----------------------------------------------

names = {58: "Wohnzimmer", 59: "Arbeitszimmer"}

logs = list(Path(".").glob("*.log"))

for l in logs:
    with open(l, "r") as o:
        content = o.readlines()

    # datetime,id,name,temp_degC,relHum_%,absHum_gkg,pres_hPa

    d = {
        "datetime": [],
        "id": [],
        "name": [],
        "temp_degC": [],
        "relHum_%": [],
        "absHum_gkg": [],
        "pres_hPa": [],
    }

    for idx, line in enumerate(content):
        if not "{" in line or not "}" in line:
            print(f"no json format: line {idx+1} in {l}: {line}")
            continue
        # parts = line.strip().split(" - ")
        # d["datetime"].append(parts[0])

        j = json.loads(line)  # parts[1])
        d["datetime"].append(j["DT"])
        d["id"].append(j["ID"])
        d["name"].append(names[int(j["ID"])])
        d["temp_degC"].append(j["T"])
        d["relHum_%"].append(j["rH"])
        d["absHum_gkg"].append(j["aH"])
        d["pres_hPa"].append(j.get("p"))

    df = pd.DataFrame(d).sort_values("datetime").fillna(0.0)
    df["datetime"] = pd.to_datetime(df["datetime"], utc=True).dt.floor("s")
    df["timestamp"] = (df["datetime"].astype(int) / 1e9).astype(int)
    # df.index = pd.to_datetime(df["datetime"])
    # df.plot(grid=True)

    fname = df["datetime"].iloc[0].strftime("%Y%m%dZ_sensordata.csv")
    df.to_csv(l.parent / fname, sep=";", index=False)


# ----------- to influxDB compatible csv --------------------------------------

# // Syntax
# <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]

# // Example
# myMeasurement,tag1=value1,tag2=value2 fieldKey="fieldValue" 1556813561098000000

# Wohnzimmer,id=58 T=13.544 1672099442
# Wohnzimmer,id=58 rH=55.04 1672099442
