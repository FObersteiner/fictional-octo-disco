import influxdb_client
from dash import Dash, html, dcc, Input, Output, dash_table

import pandas as pd

# import plotly.express as px
import plotly.graph_objects as go


cfg = pd.read_csv("db.csv")
use_idx = 0

url = cfg["url"][use_idx]
token = cfg["token"][use_idx]
org = cfg["org"][use_idx]
bucket = cfg["bucket"][use_idx]
measurements = cfg["measurements"][use_idx].split(" ")

client = influxdb_client.InfluxDBClient(url=url, token=token, org=org)

app = Dash("Dash Demo /w influxdb")


app.layout = html.Div(
    children=[
        html.H1(children="Mehr Daten !"),
        html.Label("Letzte Abfrage:"),
        html.Div(id="output_table", children=[]),
        html.Button(
            "akualisieren", id="refresh_table", n_clicks=0, style={"margin-top": 15}
        ),
        dcc.Graph(id="time-series-chart"),
        html.Button(
            "Plot akualisieren", id="refresh_plot", n_clicks=0, style={"margin-top": 15}
        ),
    ],
    style={"padding": 10},
)


@app.callback(Output("time-series-chart", "figure"), Input("refresh_plot", "n_clicks"))
def display_time_series(param):
    meas_filter = (
        f'r["_measurement"] == "{measurements[0]}"'
        if len(measurements) == 0
        else " or ".join(f'r["_measurement"] == "{m}"' for m in measurements)
    )
    query = f"""from(bucket: "{bucket}")
     |> range(start: -1h)
     |> filter(fn: (r) => {meas_filter})
     |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")"""

    dfs = client.query_api().query_data_frame(org=org, query=query)
    if not isinstance(dfs, list):  # might only be one df returned
        dfs = [dfs]
    for df in dfs:
        df["_time"] = pd.to_datetime(df["_time"]).dt.tz_convert("Europe/Berlin")

    params = ("T", "rH", "aH", "p")

    p = params[0]

    fig = go.Figure()

    for df in dfs:
        if p in df.columns:
            fig.add_scatter(
                x=df["_time"], y=df[p], mode="lines", name=df["_measurement"][0]
            )

    fig.update_xaxes(title_text="<b>Zeit</b>")
    fig.update_yaxes(title_text=f"<b>{p}</b>")
    fig.update_layout(
        legend=dict(yanchor="top", y=0.99, xanchor="left", x=0.01),
        xaxis=dict(tickformat="%H:%M"),
        # title_text=f"{p}",
        template="plotly",
    )

    return fig


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
    df = pd.concat(dfs).drop(["result", "table", "_start", "_stop", "id"], axis=1)

    # rename and reorder
    df = df.rename(columns={"_time": "Zeit", "_measurement": "Wo"})

    # parse to datetime and localize
    df["Zeit"] = (
        pd.to_datetime(df["Zeit"])
        .dt.tz_convert("Europe/Berlin")
        .dt.strftime("%d.%m.%Y %H:%M:%S")
    )

    # round values, add units
    df["T"] = df["T"].round(2).astype(str) + " °C"
    df["aH"] = df["aH"].round(2).astype(str) + " g/kg"
    df["rH"] = df["rH"].round(2).astype(str) + " %"
    df["p"] = df["p"].round(2).astype(str) + " hPa"
    df["p"][df["p"].str.contains("nan")] = "N/A"

    container = (
        dash_table.DataTable(
            data=df.to_dict("records"),
            columns=[{"name": i, "id": i} for i in df.columns],
            style_cell_conditional=[
                {"if": {"column_id": c}, "textAlign": "left"} for c in ["Zeit", "Wo"]
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
    app.run_server(port=18093, debug=True)
