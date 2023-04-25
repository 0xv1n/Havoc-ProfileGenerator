package main

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

var monacoFontData []byte

var monacoFont = fyne.StaticResource{
	StaticName:    "monaco.ttf",
	StaticContent: monacoFontData,
}

// Define Dracula Theme
type draculaTheme struct{}

var _ fyne.Theme = (*draculaTheme)(nil)

func (d draculaTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 40, G: 42, B: 54, A: 255}
	case theme.ColorNameButton:
		return color.RGBA{R: 68, G: 71, B: 90, A: 255}
	case theme.ColorNameDisabled:
		return color.RGBA{R: 98, G: 114, B: 164, A: 255}
	case theme.ColorNameHover:
		return color.RGBA{R: 255, G: 121, B: 198, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 98, G: 114, B: 164, A: 255}
	case theme.ColorNamePrimary:
		return color.RGBA{R: 80, G: 250, B: 123, A: 255}
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 68, G: 71, B: 90, A: 255}
	case theme.ColorNameShadow:
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	default:
		return theme.DarkTheme().Color(name, variant)
	}
}

func (d draculaTheme) Font(style fyne.TextStyle) fyne.Resource {
	monacoFontResource, err := fyne.LoadResourceFromPath("Monaco.ttf")
	if err != nil {
		return theme.DarkTheme().Font(style) // Fallback to the default font if there's an issue loading the Monaco font
	}
	return monacoFontResource
}

func (d draculaTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}

func (d draculaTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}

type Config struct {
	Teamserver Teamserver `hcl:"Teamserver,block"`
	Operators  []Operator `hcl:"user,block"`
	Listeners  []Listener `hcl:"listener,block"`
	Service    *Service   `hcl:"Service,block"`
	Demon      Demon      `hcl:"Demon,block"`
	Webhook    Webhook    `hcl:"Webhook,block"`
}

type Webhook struct {
	Discord Discord `hcl:"Discord,block"`
}

type Discord struct {
	Url       string `hcl:"Url,attr"`
	AvatarUrl string `hcl:"AvatarUrl,attr"`
	User      string `hcl:"User,attr"`
}

type Teamserver struct {
	Host  string `hcl:"Host,attr"`
	Port  int    `hcl:"Port,attr"`
	Build Build  `hcl:"Build,block"`
}
type Build struct {
	Compiler64 string `hcl:"Compiler64,attr"`
	Nasm       string `hcl:"Nasm,attr"`
}

type Operator struct {
	Name     string `hcl:"name,label"`
	Password string `hcl:"Password,attr"`
}
type Service struct {
	Endpoint string `hcl:"Endpoint,attr"`
	Password string `hcl:"Password,attr"`
}

type Demon struct {
	Sleep              int       `hcl:"Sleep,attr"`
	Jitter             int       `hcl:"Jitter,attr"`
	TrustXForwardedFor bool      `hcl:"TrustXForwardedFor,attr"`
	Injection          Injection `hcl:"Injection,block"`
}

type Injection struct {
	Spawn64 string `hcl:"Spawn64,attr"`
	Spawn32 string `hcl:"Spawn32,attr"`
}

type Listener struct {
	Type         string   `hcl:"-"`
	Name         string   `hcl:"-"`
	KillDate     string   `hcl:"KillDate,attr,omitempty"`
	WorkingHours string   `hcl:"WorkingHours,attr,omitempty"`
	Hosts        []string `hcl:"Hosts,attr,omitempty" cty:"list(string)"`
	HostBind     string   `hcl:"HostBind,attr,omitempty"`
	HostRotation string   `hcl:"HostRotation,attr,omitempty"`
	PortBind     int      `hcl:"PortBind,attr,omitempty"`
	PortConn     int      `hcl:"PortConn,attr,omitempty"`
	UserAgent    string   `hcl:"UserAgent,attr,omitempty"`
	Headers      []string `hcl:"Headers,attr,omitempty" cty:"list(string)"`
	Uris         []string `hcl:"Uris,attr,omitempty" cty:"list(string)"`
	Secure       bool     `hcl:"Secure,attr,omitempty"`
	Response     string   `hcl:"Response,attr,omitempty"`
	PipeName     string   `hcl:"PipeName,attr,omitempty"`
}

func main() {
	a := app.New()
	a.Settings().SetTheme(&draculaTheme{})
	w := a.NewWindow("Havoc Profile Generator")
	w.Resize(fyne.NewSize(600, w.Canvas().Size().Height))
	config := Config{}
	form := &widget.Form{}

	addOperatorButton := &widget.Button{}
	addListenerButton := &widget.Button{}
	saveListenerButton := &widget.Button{}
	cancelButton := &widget.Button{}
	saveButton := &widget.Button{}

	filename := "profile.yaotl"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Set default values for the config
		config = Config{
			Teamserver: Teamserver{
				Host: "0.0.0.0",
				Port: 40056,
				Build: Build{
					Compiler64: "data/x86_64-w64-mingw32-cross/bin/x86_64-w64-mingw32-gcc",
					Nasm:       "/usr/bin/nasm",
				},
			},
			Operators: []Operator{},
			Service:   nil,
			Demon: Demon{
				Sleep:              30,
				Jitter:             30,
				TrustXForwardedFor: false,
				Injection: Injection{
					Spawn64: "C:\\Windows\\System32\\notepad.exe",
					Spawn32: "C:\\Windows\\SysWOW64\\notepad.exe",
				},
			},
		}
	} else { // TODO: this code is for future editing of existing profiles, it's broken now
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Println("Error reading yaotl file:", err)
			os.Exit(1)
		}

		file, diags := hclsyntax.ParseConfig(content, filename, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			fmt.Println("Error parsing yaotl file:", diags.Error())
			os.Exit(1)
		}
		err = gohcl.DecodeBody(file.Body, nil, &config)
		if err != nil {
			fmt.Println("Error decoding yaotl file:", err)
			os.Exit(1)
		}
	}
	// Creating UI Elements
	profileNameEntry := widget.NewEntry()
	hostEntry := widget.NewEntry()
	hostEntry.SetText(config.Teamserver.Host)
	portEntry := widget.NewEntry()
	portEntry.SetText(fmt.Sprintf("%d", config.Teamserver.Port))

	compiler64Entry := widget.NewEntry()
	compiler64Entry.SetText(config.Teamserver.Build.Compiler64)
	nasmEntry := widget.NewEntry()
	nasmEntry.SetText(config.Teamserver.Build.Nasm)

	listenerTypeSelect := widget.NewSelect([]string{"Http", "Https", "Smb"}, nil)
	listenerNameEntry := widget.NewEntry()
	listenerNameEntry.SetPlaceHolder("Required")
	hostsEntry := widget.NewEntry()
	hostsEntry.SetPlaceHolder("Optional: Comma-separated hosts.")
	portBindEntry := widget.NewEntry()
	portBindEntry.SetPlaceHolder("Required: 8080")
	userAgentEntry := widget.NewEntry()
	userAgentEntry.SetPlaceHolder("Required")
	urisEntry := widget.NewEntry()
	urisEntry.SetPlaceHolder("Optional: /cat.png,/a.gif,etc")
	headersEntry := widget.NewEntry()
	headersEntry.SetPlaceHolder("Optional: Content-type: text/plain, X-IsHavoc: true, etc")
	responseEntry := widget.NewEntry()
	responseEntry.SetPlaceHolder("Optional")
	pipeNameEntry := widget.NewEntry()
	pipeNameEntry.SetPlaceHolder("Required: pipe_name")
	killDateEntry := widget.NewEntry()
	killDateEntry.SetPlaceHolder("Optional: 2006-01-02 15:04:05")
	workingHoursEntry := widget.NewEntry()
	workingHoursEntry.SetPlaceHolder("e.g. 8:00-17:00")
	hostBindEntry := widget.NewEntry()
	hostBindEntry.SetPlaceHolder("Required")
	portConnEntry := widget.NewEntry()
	portConnEntry.SetPlaceHolder("Optional")
	secureEntry := widget.NewCheck("", nil)
	hostRotationEntry := widget.NewSelect([]string{"random", "round-robin"}, nil)
	hostRotationEntry.SetSelected("round-robin")

	// TODO: Maybe more UI stuff?
	listenerForm := &widget.Form{}

	cancelButton = widget.NewButton("Cancel", func() {
		// Switch back to the main form when the Cancel button is clicked
		w.SetContent(container.NewVBox(
			form,
			addOperatorButton,
			addListenerButton,
			saveButton,
		))
	})

	saveListenerButton = widget.NewButton("Save Listener", func() {
		listenerType := listenerTypeSelect.Selected
		listenerName := listenerNameEntry.Text

		newListener := Listener{
			Type: listenerType,
			Name: listenerName,
		}

		if listenerType == "Http" || listenerType == "Https" {
			newListener.KillDate = killDateEntry.Text
			newListener.WorkingHours = workingHoursEntry.Text
			hosts := strings.Split(hostsEntry.Text, ",")
			newListener.Hosts = make([]string, len(hosts))
			for i, h := range hosts {
				newListener.Hosts[i] = strings.TrimSpace(h)
			}
			newListener.HostBind = hostBindEntry.Text
			newListener.HostRotation = hostRotationEntry.Selected
			port_b, err := strconv.Atoi(portBindEntry.Text)
			if err != nil {
				// handle error
			}
			port_c, err := strconv.Atoi(portConnEntry.Text)
			if err != nil {
				// handle error
			}
			newListener.PortBind = port_b
			newListener.PortConn = port_c
			newListener.UserAgent = userAgentEntry.Text

			headers := strings.Split(headersEntry.Text, ",")
			newListener.Headers = make([]string, len(headers))
			for i, h := range headers {
				newListener.Headers[i] = strings.TrimSpace(h)
			}

			uris := strings.Split(urisEntry.Text, ",")
			newListener.Uris = make([]string, len(uris))
			for i, h := range uris {
				newListener.Uris[i] = strings.TrimSpace(h)
			}
			newListener.Secure = secureEntry.Checked
			newListener.Response = responseEntry.Text
		} else if listenerType == "Smb" {
			newListener.PipeName = pipeNameEntry.Text
		}

		config.Listeners = append(config.Listeners, newListener)

		// Switch back to the main form when the Save Listener button is clicked
		w.SetContent(container.NewVBox(
			form,
			addOperatorButton,
			addListenerButton,
			saveButton,
		))
	})

	addListenerButton = widget.NewButton("Add Listener", func() {
		// Switch to the listener form when the Add Listener button is clicked
		w.SetContent(container.NewVBox(
			listenerForm,
			cancelButton,
			saveListenerButton,
		))
	})

	listenerTypeSelect = widget.NewSelect([]string{"Http", "Https", "Smb"}, nil)
	listenerNameEntry = widget.NewEntry()

	// Function to update the form based on the selected listener type
	updateListenerForm := func(listenerType string) {
		listenerForm.Items = nil
		listenerForm.Append("Listener Type:", listenerTypeSelect)
		listenerForm.Append("Listener Name:", listenerNameEntry)

		if listenerType == "Http" || listenerType == "Https" {
			listenerForm.Append("KillDate:", killDateEntry)
			listenerForm.Append("WorkingHours:", workingHoursEntry)
			listenerForm.Append("Hosts:", hostsEntry)
			listenerForm.Append("HostBind:", hostBindEntry)
			listenerForm.Append("HostRotation:", hostRotationEntry)
			listenerForm.Append("PortBind:", portBindEntry)
			listenerForm.Append("PortConn:", portConnEntry)
			listenerForm.Append("UserAgent:", userAgentEntry)
			listenerForm.Append("Headers:", headersEntry)
			listenerForm.Append("Uris:", urisEntry)
			listenerForm.Append("Secure:", secureEntry)
			listenerForm.Append("Response:", responseEntry)
		} else if listenerType == "Smb" {
			listenerForm.Append("PipeName:", pipeNameEntry)
		}
		listenerForm.Refresh()
	}

	// Initialize the form with the default listener type
	updateListenerForm("Http")

	// Update the form when the listener type changes
	listenerTypeSelect.OnChanged = func(listenerType string) {
		updateListenerForm(listenerType)
	}

	operatorEntries := []struct {
		name     *widget.Entry
		password *widget.Entry
	}{}
	for _, op := range config.Operators {
		operatorName := widget.NewEntry()
		operatorName.SetText(op.Name)
		operatorPassword := widget.NewEntry()
		operatorPassword.SetText(op.Password)

		operatorEntries = append(operatorEntries, struct {
			name     *widget.Entry
			password *widget.Entry
		}{
			name:     operatorName,
			password: operatorPassword,
		})
	}

	sleepEntry := widget.NewEntry()
	sleepEntry.SetText(fmt.Sprintf("%d", config.Demon.Sleep))
	jitterEntry := widget.NewEntry()
	jitterEntry.SetText(fmt.Sprintf("%d", config.Demon.Jitter))

	trustXForwardedForEntry := widget.NewCheck("", nil)
	trustXForwardedForEntry.SetChecked(config.Demon.TrustXForwardedFor)

	spawn64Entry := widget.NewEntry()
	spawn64Entry.SetText(config.Demon.Injection.Spawn64)
	spawn32Entry := widget.NewEntry()
	spawn32Entry.SetText(config.Demon.Injection.Spawn32)

	addOperatorButton = widget.NewButton("Add Operator", func() {
		operatorName := widget.NewEntry()
		operatorPassword := widget.NewEntry()
		operatorEntries = append(operatorEntries, struct {
			name     *widget.Entry
			password *widget.Entry
		}{
			name:     operatorName,
			password: operatorPassword,
		})
		form.Append("User Name:", operatorName)
		form.Append("User Password:", operatorPassword)
		form.Refresh()
	})

	saveButton = widget.NewButton("Save", func() {

		f := hclwrite.NewEmptyFile()
		rootBody := f.Body()

		teamserverBlock := rootBody.AppendNewBlock("Teamserver", nil)
		teamserverBody := teamserverBlock.Body()
		teamserverBody.SetAttributeValue("Host", cty.StringVal(hostEntry.Text))
		port, _ := strconv.ParseInt(portEntry.Text, 10, 64)
		teamserverBody.SetAttributeValue("Port", cty.NumberIntVal(port))

		buildBlock := teamserverBody.AppendNewBlock("Build", nil)
		buildBody := buildBlock.Body()
		buildBody.SetAttributeValue("Compiler64", cty.StringVal(compiler64Entry.Text))
		buildBody.SetAttributeValue("Nasm", cty.StringVal(nasmEntry.Text))

		operatorsBlock := rootBody.AppendNewBlock("Operators", nil)
		operatorsBody := operatorsBlock.Body()

		for _, op := range operatorEntries {
			operatorBlock := operatorsBody.AppendNewBlock("user", []string{op.name.Text})
			operatorBody := operatorBlock.Body()
			operatorBody.SetAttributeValue("Password", cty.StringVal(op.password.Text))
		}

		// Listener Work
		listenersBlock := rootBody.AppendNewBlock("Listeners", nil)
		listenersBody := listenersBlock.Body()

		for _, listener := range config.Listeners {
			listenerTypeBlock := listenersBody.AppendNewBlock(listener.Type, nil)
			listenerTypeBody := listenerTypeBlock.Body()
			listenerTypeBody.SetAttributeValue("Name", cty.StringVal(listener.Name))
			if listener.Type == "Smb" {
				listenerTypeBody.SetAttributeValue("PipeName", cty.StringVal(listener.PipeName))
			} else {
				if listener.KillDate != "" {
					listenerTypeBody.SetAttributeValue("KillDate", cty.StringVal(listener.KillDate))
				}
				if listener.WorkingHours != "" {
					listenerTypeBody.SetAttributeValue("WorkingHours", cty.StringVal(listener.WorkingHours))
				}
				if len(listener.Hosts) > 0 && listener.Hosts[0] != "" {
					hosts := make([]cty.Value, len(listener.Hosts))
					for i, host := range listener.Hosts {
						hosts[i] = cty.StringVal(host)
					}
					listenerTypeBody.SetAttributeValue("Hosts", cty.ListVal(hosts))
				}
				if listener.HostBind != "" {
					listenerTypeBody.SetAttributeValue("HostBind", cty.StringVal(listener.HostBind))
				}
				if listener.HostRotation != "" {
					listenerTypeBody.SetAttributeValue("HostRotation", cty.StringVal(listener.HostRotation))
				}
				listenerTypeBody.SetAttributeValue("PortBind", cty.NumberIntVal(int64(listener.PortBind)))
				listenerTypeBody.SetAttributeValue("PortConn", cty.NumberIntVal(int64(listener.PortConn)))
				if listener.UserAgent != "" {
					listenerTypeBody.SetAttributeValue("UserAgent", cty.StringVal(listener.UserAgent))
				}
				if len(listener.Headers) > 0 && listener.Headers[0] != "" {
					headers := make([]cty.Value, len(listener.Headers))
					for i, header := range listener.Headers {
						headers[i] = cty.StringVal(header)
					}
					listenerTypeBody.SetAttributeValue("Headers", cty.ListVal(headers))
				}
				if len(listener.Uris) > 0 && listener.Uris[0] != "" {
					uris := make([]cty.Value, len(listener.Uris))
					for i, uri := range listener.Uris {
						uris[i] = cty.StringVal(uri)
					}
					listenerTypeBody.SetAttributeValue("Uris", cty.ListVal(uris))
				}
				listenerTypeBody.SetAttributeValue("Secure", cty.BoolVal(secureEntry.Checked))
				if listener.Response != "" {
					listenerTypeBody.SetAttributeValue("Response", cty.StringVal(listener.Response))
				}
			}
		}

		demonBlock := rootBody.AppendNewBlock("Demon", nil)
		demonBody := demonBlock.Body()
		sleep, _ := strconv.ParseInt(sleepEntry.Text, 10, 64)
		demonBody.SetAttributeValue("Sleep", cty.NumberIntVal(sleep))
		jitter, _ := strconv.ParseInt(jitterEntry.Text, 10, 64)
		demonBody.SetAttributeValue("Jitter", cty.NumberIntVal(jitter))
		demonBody.SetAttributeValue("TrustXForwardedFor", cty.BoolVal(trustXForwardedForEntry.Checked))
		injectionBlock := demonBody.AppendNewBlock("Injection", nil)
		injectionBody := injectionBlock.Body()
		injectionBody.SetAttributeValue("Spawn64", cty.StringVal(spawn64Entry.Text))
		injectionBody.SetAttributeValue("Spawn32", cty.StringVal(spawn32Entry.Text))

		if profileNameEntry.Text == "" {
			filename = "profile.yaotl"
		} else {
			filename = profileNameEntry.Text + ".yaotl"
		}
		err := ioutil.WriteFile(filename, f.Bytes(), 0644)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", "Profile file saved successfully!", w)
		}
	})

	form.Items = []*widget.FormItem{
		{Text: "Profile Name:", Widget: profileNameEntry},
		{Text: "Teamserver Host:", Widget: hostEntry},
		{Text: "Teamserver Port:", Widget: portEntry},
		{Text: "Compiler64:", Widget: compiler64Entry},
		{Text: "NASM:", Widget: nasmEntry},
	}

	// TODO: add more in the event of profile malleability changes
	for _, op := range operatorEntries {
		form.Append("User Name:", op.name)
		form.Append("User Password:", op.password)
	}

	form.Append("Demon Sleep:", sleepEntry)
	form.Append("Demon Jitter:", jitterEntry)
	form.Append("TrustXForwardedFor:", trustXForwardedForEntry)
	form.Append("Injection Spawn64:", spawn64Entry)
	form.Append("Injection Spawn32:", spawn32Entry)

	w.SetContent(container.NewVBox(
		form,
		addOperatorButton,
		addListenerButton,
		saveButton,
	))

	w.ShowAndRun()
}
