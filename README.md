# retroterm

Try it here : https://retroterm.de

A browser based retro terminal with a backend written in go.
This thing was created as an experiment to create a project using only AI as coder.
Thus 99% of the code was created by various models of Github Copilot.
(That was a nightmare !)

The frontend creates a Green Monitor used as a terminal.
All but the I/O stuff is handled in the backend.

Features :
Guests and logged in users
Guests have a virtual filesystem in ram (gets lost), users get one in the sqlite database
Virtual OS with some commands
Basic Interpreter
A chat application with a very special 80's personality connected to the Deepseek API
SAM speech synthesizer usable in Basic
Basic has some sound, sprite, graphics and vectorgraphics commands 
