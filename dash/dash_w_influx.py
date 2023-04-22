import math
from time import monotonic as ticker
import warnings

from dash import Dash, html, dcc, Input, Output, dash_table
import influxdb_client
from influxdb_client.client.warnings import MissingPivotFunction
import pandas as pd
import plotly.express as px
from plotly.subplots import make_subplots
import tomli as toml


warnings.simplefilter("ignore", MissingPivotFunction)

with open("config.toml", "rb") as fp:
    cfg = toml.load(fp)

try:
    colors = getattr(px.colors.qualitative, cfg["app"]["colors"])
except AttributeError:
    colors = px.colors.qualitative.Plotly

url = cfg["db"]["url"]
token = cfg["db"]["token"]
org = cfg["db"]["org"]
bucket = cfg["db"]["bucket"]
measurements = cfg["db"]["measurements"]
params = cfg["app"]["parameters"]

client = influxdb_client.InfluxDBClient(url=url, token=token, org=org)

app = Dash("solltIchLueften")

app.layout = html.Div(
    children=[
        html.Div(
            [
                html.H2(children="Mehr Daten !"),
                # html.Label("Letzte Werte:"),
                html.Div(id="output_table", children=[]),
                html.Button(
                    "Tabelle akualisieren",
                    id="refresh_table",
                    n_clicks=0,
                    style={"margin-top": 15},
                ),
            ]
        ),
        html.Div(
            [
                html.H2(children="Plots !"),
                html.Button(
                    "Plots akualisieren",
                    id="refresh_plot",
                    n_clicks=0,
                    style={"margin-top": 5, "margin-bottom": 15},
                ),
                dcc.Dropdown(
                    id="timeframe",
                    options=[
                        {"label": "letzte Stunde", "value": "-1h"},
                        {"label": "6 Stunden zurueck", "value": "-6h"},
                        {"label": "letzter Tag", "value": "-1d"},
                        {"label": "letzte Woche", "value": "-1w"},
                    ],
                    value="-1h",
                    clearable=False,
                ),
            ]
        ),
    ]
)

# add the graphs, one for each specified parameter
app.layout.children += [html.Div([dcc.Graph(id=f"plot_{p}")]) for p in params]


def get_ranges(data, params):
    """Plot y-ranges"""
    r = {}
    for p in params:
        if p == "T":  # T: draussen ist auf zweiter y-Achse
            m = (data["_field"] == p) & (data["_measurement"] == "Draussen")
            try:
                r[p + "_draussen"] = [
                    math.floor(data["_value"][m].min()),
                    math.ceil(data["_value"][m].max()),
                ]
            except ValueError:
                r[p + "_draussen"] = [0, 0]
            m = (data["_field"] == p) & (data["_measurement"] != "Draussen")
            try:
                r[p] = [
                    math.floor(data["_value"][m].min()),
                    math.ceil(data["_value"][m].max()),
                ]
            except ValueError:
                r[p] = [0, 0]
        else:
            m = data["_field"] == p
            try:
                r[p] = [
                    math.floor(data["_value"][m].min()),
                    math.ceil(data["_value"][m].max()),
                ]
            except ValueError:
                r[p] = [0, 0]
            r[p + "_draussen"] = r[p]  # selbe Range für drinnen und draussen

        if p in ("p", "rH"):
            r[p + "_draussen"] = [r[p + "_draussen"][0] - 1, r[p + "_draussen"][1] + 1]
            r[p] = [r[p][0] - 1, r[p][1] + 1]
        if p in ("T",):
            r[p + "_draussen"] = [
                r[p + "_draussen"][0] - 0.5,
                r[p + "_draussen"][1] + 0.5,
            ]
            r[p] = [r[p][0] - 0.5, r[p][1] + 0.5]

    return r


@app.callback(
    [Output(f"plot_{p}", "figure") for p in params],
    [Input("refresh_plot", "n_clicks"), Input("timeframe", "value")],
)
def display_time_series(n, timeframe):
    """Plots ?!"""
    meas_filter = (
        f'r["_measurement"] == "{measurements[0]}"'
        if len(measurements) == 0
        else " or ".join(f'r["_measurement"] == "{m}"' for m in measurements)
    )

    query = f"""from(bucket: "{bucket}")
     |> range(start: {timeframe})
     |> filter(fn: (r) => {meas_filter})"""
    # |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")"""

    # pivot does not work correctly for more than two parameters
    # bug in 'query_data_frame' ?

    # t = ticker()
    data = client.query_api().query_data_frame(org=org, query=query)
    if isinstance(data, list):
        data = pd.concat(data)
    # print(f"query df: {ticker()-t:.4f} s")

    # t = ticker()
    data["_time"] = data["_time"].dt.tz_convert("Europe/Berlin")
    # print(f"tz_convert: {ticker()-t:.4f} s")

    # t = ticker()
    ranges = get_ranges(data, params)
    # print(f"calculate plot ranges: {ticker()-t:.4f} s")

    t = ticker()
    figs = []

    for p in params:  # loop parameters
        fig = make_subplots(specs=[[{"secondary_y": True}]])

        for idx_meas, m in enumerate(measurements):
            w_df = data["_measurement"] == m

            if not w_df.any():  # no data for defined measurement?
                continue

            df = data[w_df]

            if p in df._field.values:
                m_p = df._field.values == p
                fig.add_scatter(
                    x=df["_time"][m_p],
                    y=df["_value"][m_p],
                    mode="lines",
                    name=m,
                    secondary_y=m == "Draussen" and p == "T",
                    line=dict(color=colors[idx_meas % len(colors)]),
                )

        title = p if (longname := cfg["long_names"].get(p)) is None else longname

        fig.update_layout(
            legend=dict(
                yanchor="top",
                y=0.99,
                xanchor="left",
                x=0.01,
                bgcolor="rgba(255,255,255,0.7)",
            ),
            margin=dict(l=55, r=40, t=80, b=80),
            title_text=title,
            template=cfg["app"]["plotly_theme"],
        )

        if (ylabel := cfg["units"].get(p)) is not None:
            fig.update_yaxes(
                range=ranges[p], title_text=f"<b>{ylabel}</b>", secondary_y=False
            )
            fig.update_yaxes(
                range=ranges[p + "_draussen"], title_text="(draussen)", secondary_y=True
            )

        figs.append(fig)

    figs[-1].update_xaxes(title_text="<b>Zeit</b>")
    print(f"make plots: {ticker()-t:.4f} s")

    return figs


@app.callback(
    [
        Output(component_id="output_table", component_property="children"),
    ],
    [
        Input("refresh_table", "n_clicks"),
    ],
)
def update_table(_):
    """Tabelle !"""
    meas_filter = (
        f'r["_measurement"] == "{measurements[0]}"'
        if len(measurements) == 0
        else " or ".join(f'r["_measurement"] == "{m}"' for m in measurements)
    )

    query = f"""from(bucket: "{bucket}")
    |> range(start: -5m)
    |> tail(n: 1)
    |> filter(fn: (r) => {meas_filter})"""
    # |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")"""

    data = client.query_api().query_data_frame(org=org, query=query)
    # sometimes, this is returned as a list of dfs ?!
    if isinstance(data, list):
        data = pd.concat(data)

    data["_time"] = data["_time"].dt.tz_convert("Europe/Berlin")

    d = {"Wo": [], "Wann": [], "Was": [], "Wert": []}
    for m in measurements:
        w = data["_measurement"] == m
        if not w.any():  # there might be no data for set measurement
            continue
        df = data[w]
        for p in params:
            if p in df._field.values:
                d["Wo"].append(m)
                d["Wann"].append(df["_time"].iloc[0].strftime("%d.%m.%Y %H:%M:%S"))
                d["Was"].append(p)
                d["Wert"].append(
                    f"%{cfg['unit_formats'][p]}"
                    % (df["_value"][df["_field"] == p]).iloc[0]
                )
    df = pd.DataFrame(d)

    container = (
        dash_table.DataTable(
            data=df.to_dict("records"),
            columns=[{"name": i, "id": i} for i in df.columns],
            style_cell_conditional=[
                {"if": {"column_id": c}, "textAlign": "left"} for c in ["Wo", "Wann"]
            ],
            style_data_conditional=[
                {
                    "if": {"filter_query": "{{Wo}} = {}".format(m)},
                    # "background-color": colors[i],
                    "color": colors[i],
                }
                for i, m in enumerate(measurements)
            ],
            style_header={
                "backgroundColor": "white",
                "color": "black",
                "font_size": "14px",
                "fontWeight": "bold",
                "border": "2px solid black",
            },
            style_data={
                "backgroundColor": "white",
                "font_size": "13px",
                "fontWeight": "bold",
                "border": "1px solid black",
            },
        ),
    )
    return container


if __name__ == "__main__":
    app.run_server(
        host=cfg["app"]["host"], port=cfg["app"]["port"], debug=cfg["app"]["debug"]
    )

# scp -r /home/floo/Code/Mixed/fictional-octo-disco/dash/ floo@192.168.0.107:/home/floo/Documents/
