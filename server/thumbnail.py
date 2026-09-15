from io import BytesIO

import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt
from matplotlib.ticker import MaxNLocator, PercentFormatter

from charts import _padded_xlim, _percent_axis

IMAGE_WIDTH_INCHES = 12.0
IMAGE_HEIGHT_INCHES = 6.3
IMAGE_DPI = 100

BACKGROUND = "#0a0e1a"
BRAND = "#7c9eff"
HEADING = "#ffffff"
MUTED = "#8ba3c9"
POSITIVE = "#4ade80"
NEGATIVE = "#f87171"
GRID = "#16203a"
SPINE = "#22304d"

LONG_NAME_LENGTH = 12
LONG_HEADLINE_LENGTH = 7


def render_thumbnail_png(username, card):
    fig = plt.figure(figsize=(IMAGE_WIDTH_INCHES, IMAGE_HEIGHT_INCHES), dpi=IMAGE_DPI)
    fig.patch.set_facecolor(BACKGROUND)

    name = str(username)
    name_size = 52 if len(name) <= LONG_NAME_LENGTH else 36
    headline_size = 72 if len(card.headline) <= LONG_HEADLINE_LENGTH else 44
    headline_color = POSITIVE if card.positive else NEGATIVE

    fig.text(0.06, 0.84, "ASTROSWARM", fontsize=26, color=BRAND, weight="bold")
    fig.text(0.06, 0.66, name, fontsize=name_size, color=HEADING, weight="bold")
    fig.text(0.06, 0.55, card.level_name, fontsize=20, color=MUTED)
    fig.text(0.06, 0.30, card.headline, fontsize=headline_size, color=headline_color, weight="bold")
    fig.text(0.06, 0.20, card.caption, fontsize=16, color=MUTED)

    if card.trend:
        _draw_trend(fig, card)
    elif card.facts:
        _draw_facts(fig, card.facts)

    return _save(fig)


def _draw_trend(fig, card):
    ax = fig.add_axes([0.58, 0.2, 0.36, 0.58])
    ax.set_facecolor(BACKGROUND)
    trials = [trial for trial, _ in card.trend]
    rates = [rate for _, rate in card.trend]
    ax.plot(trials, rates, color=POSITIVE, linewidth=3, marker="o" if len(trials) == 1 else None)
    _percent_axis(ax)
    _padded_xlim(ax, trials)
    ax.yaxis.set_major_formatter(PercentFormatter(decimals=0))
    ax.xaxis.set_major_locator(MaxNLocator(integer=True))
    ax.set_title(card.trend_title, color=MUTED, fontsize=14)
    ax.set_xlabel("Trial", color=MUTED, fontsize=12)
    ax.tick_params(colors=MUTED)
    for spine in ax.spines.values():
        spine.set_color(SPINE)
    ax.grid(True, color=GRID)


def _draw_facts(fig, facts):
    top = 0.74
    spacing = 0.19
    for index, (label, value) in enumerate(facts):
        y = top - index * spacing
        fig.text(0.62, y, label, fontsize=16, color=MUTED)
        fig.text(0.62, y - 0.08, value, fontsize=30, color=HEADING, weight="bold")


def _save(fig):
    buffer = BytesIO()
    fig.savefig(buffer, format="png", dpi=IMAGE_DPI, facecolor=fig.get_facecolor())
    plt.close(fig)
    buffer.seek(0)
    return buffer.getvalue()
