# Security

## Reporting a vulnerability

Email **justin@justinstimatze.com** with `germline-security` in the subject.
Please do not file public issues for suspected vulnerabilities.

## Threat model

germline runs locally and sends nothing over the network. It reads the
project tree it is pointed at and the decision record that project names,
either its own `decisions/` directory or an external store given by `-store`
or `git config winze.store`. The record is parsed as Go source and never
built or executed.

It executes two kinds of program, both named by the operator: the replay
subjects passed to `germline replay` (`-was` and `-now`), run as argument
lists without a shell, and `git`, to read the committed manifest. The
optional `replaygate` Stop hook runs `make replay` in the repository where
it is installed. Treat a replay subject and a `Makefile` with the same trust
as any program you run by hand.
