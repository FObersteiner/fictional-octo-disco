import warnings
from influxdb_client.client.warnings import MissingPivotFunction

warnings.simplefilter("ignore", MissingPivotFunction)

import influxdb_client
from dash import Dash, html, dcc, Input, Output, dash_table

import pandas as pd
import tomli as toml

import plotly.express as px

from plotly.subplots import make_subplots


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
                html.H1(children="Mehr Daten !"),
                html.Label("Letzte Werte:"),
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
                        {"label": "letzten 3 Stunden", "value": "-3h"},
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


@app.callback(
    [Output(f"plot_{p}", "figure") for p in params],
    [Input("refresh_plot", "n_clicks"), Input("timeframe", "value")],
)
def display_time_series(n, timeframe):
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

    data = client.query_api().query_data_frame(org=org, query=query)
    data["_time"] = pd.to_datetime(data["_time"]).dt.tz_convert("Europe/Berlin")

    figs = []
    for _, p in enumerate(params):  # loop parameters
        fig = make_subplots(specs=[[{"secondary_y": True}]])
        for idx_meas, (name, df) in enumerate(
            data.groupby("_measurement")
        ):  # loop data sources
            if p in df._field.values:
                m = df._field.values == p
                fig.add_scatter(
                    x=df["_time"][m],
                    y=df["_value"][m],
                    mode="lines",
                    name=name,
                    secondary_y=name == "Draussen" and p == "T",
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
            fig.update_yaxes(title_text=f"<b>{ylabel}</b>", secondary_y=False)
            fig.update_yaxes(title_text="(draussen)", secondary_y=True)

        figs.append(fig)

    figs[-1].update_xaxes(title_text="<b>Zeit</b>")

    return figs


@app.callback(
    [
        Output(component_id="output_table", component_property="children"),
    ],
    [
        Input("refresh_table", "n_clicks"),
    ],
)
def update_table(n_clicks):
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
    data["_time"] = pd.to_datetime(data["_time"]).dt.tz_convert("Europe/Berlin")

    d = {"Wo": [], "Wann": [], "Was": [], "Wert": []}
    for name, df in data.groupby("_measurement"):
        for p in params:
            if p in df._field.values:
                d["Wo"].append(name)  # df["_measurement"][0])
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
