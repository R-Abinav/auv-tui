package main

import (
	"fmt"
	"net"
	"strings"
	"github.com/rivo/tview"
	"golang.org/x/crypto/ssh"
)

type App struct {
	tui       *tview.Application
	logView   *tview.TextView
	mainFlex  *tview.Flex
	sshClient *ssh.Client
}

func main() {
	tui := tview.NewApplication()
	logView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			tui.Draw()
		})
	logView.SetBorder(true).SetTitle("Output / Logs")

	app := &App{
		tui:     tui,
		logView: logView,
	}

	menu := tview.NewList().
		AddItem("1. Connect to Jetson", "Establish an SSH connection", '1', app.showConnectForm).
		AddItem("2. Start ROS Core", "Run 'roscore' in a background session", '2', app.startRosCore).
		AddItem("3. Start Rosserial", "Run the rosserial client for the microcontroller", '3', app.startRosserial).
		AddItem("4. Run a Script", "Select and run a ROS node", '4', app.showScriptSelector).
		AddItem("Quit", "Exit the application", 'q', func() {
			if app.sshClient != nil {
				app.sshClient.Close()
			}
			tui.Stop()
		})
	menu.SetBorder(true).SetTitle("AUV Control Menu")

	app.mainFlex = tview.NewFlex().
		AddItem(menu, 0, 1, true).
		AddItem(logView, 0, 3, false)

	if err := tui.SetRoot(app.mainFlex, true).Run(); err != nil {
		panic(err)
	}
}

func (a *App) log(message string) {
	a.tui.QueueUpdateDraw(func() {
		fmt.Fprintln(a.logView, message)
	})
}

func (a *App) showConnectForm() {
	var form *tview.Form
	form = tview.NewForm().
		AddInputField("Jetson IP", "192.168.1.100", 20, nil, nil).
		AddPasswordField("Password", "", 20, '*', nil).
		AddButton("Connect", func() {
			ip := form.GetFormItem(0).(*tview.InputField).GetText()
			password := form.GetFormItem(1).(*tview.InputField).GetText()
			go a.connectToJetson(ip, password)
			a.tui.SetRoot(a.mainFlex, true)
		}).
		AddButton("Cancel", func() {
			a.tui.SetRoot(a.mainFlex, true)
		})

	form.SetBorder(true).SetTitle("Connect to Jetson").SetTitleAlign(tview.AlignLeft)
	a.tui.SetRoot(form, true)
}

func (a *App) connectToJetson(ip, password string) {
	a.log(fmt.Sprintf("[yellow]Connecting to jetson@%s...", ip))
	config := &ssh.ClientConfig{
		User: "jetson",
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", net.JoinHostPort(ip, "22"), config)
	if err != nil {
		a.log(fmt.Sprintf("[red]Failed to connect: %v", err))
		return
	}
	a.sshClient = client
	a.log("[green]Successfully connected to Jetson!")
}

func (a *App) runRemoteCommand(command string, streamOutput bool) {
	if a.sshClient == nil {
		a.log("[red]Error: Not connected to Jetson. Please connect first.")
		return
	}
	a.log(fmt.Sprintf("[yellow]Running: %s", command))
	session, err := a.sshClient.NewSession()
	if err != nil {
		a.log(fmt.Sprintf("[red]Failed to create session: %v", err))
		return
	}
	defer session.Close()

	if streamOutput {
		session.Stdout = a.logView
		session.Stderr = a.logView
		if err := session.Run(command); err != nil {
			a.log(fmt.Sprintf("[red]Command failed: %v", err))
		}
	} else {
		output, err := session.CombinedOutput(command)
		if err != nil {
			a.log(fmt.Sprintf("[red]Command failed: %s\n%s", err, string(output)))
		} else {
			a.log(fmt.Sprintf("[green]Command started successfully in the background."))
		}
	}
}

func (a *App) startRosCore() {
	cmd := "screen -dmS roscore roscore"
	a.runRemoteCommand(cmd, false)
}

func (a *App) startRosserial() {
	cmd := "screen -dmS rosserial <Enter the full command gomma>"
	a.runRemoteCommand(cmd, false)
}

func (a *App) showScriptSelector() {
	if a.sshClient == nil {
		a.log("[red]Error: Not connected to Jetson. Please connect first.")
		return
	}
	a.log("[yellow]Fetching list of scripts from ROS workspace...")
	findCmd := "find /home/jetson/catkin_ws/devel/lib -maxdepth 2 -type f -executable"
	session, err := a.sshClient.NewSession()
	if err != nil {
		a.log(fmt.Sprintf("[red]Failed to create session: %v", err))
		return
	}
	defer session.Close()

	output, err := session.CombinedOutput(findCmd)
	if err != nil {
		a.log(fmt.Sprintf("[red]Failed to find scripts: %s\n%s", err, string(output)))
		return
	}

	scripts := strings.Split(strings.TrimSpace(string(output)), "\n")
	scriptList := tview.NewList()
	scriptList.SetBorder(true).SetTitle("Select a Script to Run")

	for _, scriptPath := range scripts {
		if scriptPath == "" {
			continue
		}
		parts := strings.Split(scriptPath, "/")
		if len(parts) < 2 {
			continue
		}
		scriptName := parts[len(parts)-1]
		packageName := parts[len(parts)-2]

		func(pkg, script string) {
			listItemText := fmt.Sprintf("%s/%s", pkg, script)
			scriptList.AddItem(listItemText, scriptPath, 0, func() {
				rosrunCmd := fmt.Sprintf("rosrun %s %s", pkg, script)
				go a.runRemoteCommand(rosrunCmd, true)
				a.tui.SetRoot(a.mainFlex, true)
			})
		}(packageName, scriptName)
	}
	scriptList.AddItem("Back", "Return to main menu", 'b', func() {
		a.tui.SetRoot(a.mainFlex, true)
	})
	a.tui.SetRoot(scriptList, true)
}