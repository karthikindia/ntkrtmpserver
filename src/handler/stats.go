package handler
// hlswatch - keep track of hls viewer stats
// Copyright (C) 2017 Maximilian Pachl

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// --------------------------------------------------------------------------------------
//  imports
// --------------------------------------------------------------------------------------

import (
    "net/http"
    "time"

    "github.com/faryon93/hlswatch/state"
)


// --------------------------------------------------------------------------------------
//  type
// --------------------------------------------------------------------------------------

type streamStats struct {
    // number of current viewers
    CurrentViewers int `json:"current_viewers"`

    // uptime of stream in seconds
    Uptime int `json:"uptime"`
}


// --------------------------------------------------------------------------------------
//  http handler
// --------------------------------------------------------------------------------------

func Stats(ctx *state.State, w http.ResponseWriter, r *http.Request) {
    timeout := time.Duration(ctx.Conf.Common.ViewerTimeout) * time.Second

    // construct the response
    response := make(map[string]streamStats)
    for streamName, stream := range ctx.Streams {
        response[streamName] = streamStats{
            CurrentViewers: stream.GetCurrentViewers(timeout),
            Uptime: int(stream.GetUptime().Seconds()),
        }
    }

    Jsonify(w, response)
}
