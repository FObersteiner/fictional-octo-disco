import influxdb_client
from dash import Dash, html, dcc, Input, Output, dash_table

import pandas as pd
import tomli as toml

import plotly.express as px

# from plotly.subplots import make_subplots
import plotly.graph_objects as go

# from matplotlib import colors as mcolors
# colors = list(mcolors.CSS4_COLORS.keys())

with open("config.toml", "rb") as fp:
    cfg = toml.load(fp)

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
                html.Label("Letzte Abfrage:"),
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
                html.Button(
                    "Plots akualisieren",
                    id="refresh_plot",
                    n_clicks=0,
                    style={"margin-top": 50, "margin-bottom": 15},
                ),
                dcc.Dropdown(
                    id="timeframe",
                    options=[
                        {"label": "letzte Stunde", "value": "-1h"},
                        {"label": "-3 Stunden", "value": "-3h"},
                        {"label": "-24 Stunden", "value": "-1d"},
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
    # [Output("time-series-chart1", "figure"), Output("time-series-chart2", "figure")],
    [Output(f"plot_{p}", "figure") for p in params],
    [Input("refresh_plot", "n_clicks"), Input("timeframe", "value")],
)
def display_time_series(n, timeframe):
    print(n, timeframe)
    meas_filter = (
        f'r["_measurement"] == "{measurements[0]}"'
        if len(measurements) == 0
        else " or ".join(f'r["_measurement"] == "{m}"' for m in measurements)
    )

    query = f"""from(bucket: "{bucket}")
     |> range(start: {timeframe})
     |> filter(fn: (r) => {meas_filter})
     |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")"""

    dfs = client.query_api().query_data_frame(org=org, query=query)

    if not isinstance(dfs, list):  # might only be one df returned
        dfs = [dfs]

    for df in dfs:
        df["_time"] = pd.to_datetime(df["_time"]).dt.tz_convert("Europe/Berlin")

    figs = []
    for idx_param, p in enumerate(params):
        fig = go.Figure()
        for idx_meas, df in enumerate(dfs):
            if p in df.columns:
                fig.add_scatter(
                    x=df["_time"],
                    y=df[p],
                    mode="lines",
                    name=df["_measurement"][0],
                    # line=dict(color=colors[idx_param + idx_meas]),
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
            # xaxis=dict(tickformat="%H:%M"),
            margin=dict(l=40, r=40, t=50, b=40),
            title_text=title,
            template=cfg["app"]["theme"],
        )
        if (ylabel := cfg["units"].get(p)) is not None:
            fig.update_yaxes(title_text=f"<b>{ylabel}</b>")
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
    |> filter(fn: (r) => {meas_filter})
    |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
    """

    dfs = client.query_api().query_data_frame(org=org, query=query)
    if not isinstance(dfs, list):
        dfs = [dfs]

    d = {"Wo": [], "Wann": [], "Was": [], "Wert": []}
    params = ("T", "rH", "aH", "p")
    for df in dfs:
        for p in params:
            if p in df.columns:
                d["Wo"].append(df["_measurement"][0])
                d["Wann"].append(
                    pd.to_datetime(df["_time"])
                    .dt.tz_convert("Europe/Berlin")
                    .dt.strftime("%d.%m.%Y %H:%M:%S")[0]
                )
                d["Was"].append(p)
                d["Wert"].append(df[p][0].round(2))

    df = pd.DataFrame(d)
    container = (
        dash_table.DataTable(
            data=df.to_dict("records"),
            columns=[{"name": i, "id": i} for i in df.columns],
            style_cell_conditional=[
                {"if": {"column_id": c}, "textAlign": "left"} for c in ["Wo", "Wann"]
            ],
            style_header={
                "backgroundColor": "rgb(210, 210, 210)",
                "color": "black",
                "fontWeight": "bold",
                "border": "2px solid black",
            },
            style_data={
                "color": "black",
                "backgroundColor": "white",
                "border": "1px solid black",
            },
        ),
    )
    return container


if __name__ == "__main__":
    app.run_server(
        host=cfg["app"]["host"], port=cfg["app"]["port"], debug=cfg["app"]["debug"]
    )
