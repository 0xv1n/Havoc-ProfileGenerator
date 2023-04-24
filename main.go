package main

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"os"
	"strconv"

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
	return theme.DarkTheme().Font(style)
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
	Service    *Service   `hcl:"Service,block"`
	Demon      Demon      `hcl:"Demon,block"`
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

func main() {
	a := app.New()
	a.Settings().SetTheme(&draculaTheme{})
	w := a.NewWindow("Havoc Profile Generator")
	config := Config{}
	form := &widget.Form{}

	filename := "profile.yaotl"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Set default values for the config
		config = Config{
			Teamserver: Teamserver{
				Host: "",
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

	// TODO: Maybe more UI stuff?

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

	addOperatorButton := widget.NewButton("Add Operator", func() {
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

	saveButton := widget.NewButton("Save", func() {

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

		filename = profileNameEntry.Text + ".yaotl"
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
		saveButton,
	))

	w.ShowAndRun()
}
