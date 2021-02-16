package main

const indexTemplate = `
{{ define "index" }}
<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">

		<title>RPi IR Remote</title>

		<link rel="stylesheet"
		 href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css"
		 integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T"
		 crossorigin="anonymous">
	</head>
	<body>
		<div class="container mt-1">
			<div class="col-sm text-center">
				{{ $numRemotes := len .Remotes }}
				{{ $i := 0 }}
				{{ range $remoteName, $remote := .Remotes }}
					<p><b>{{ $remoteName }}</b></p>

					{{ range $i, $code := $remote.Code }}
						<button type="button" class="btn btn-primary" name="{{ $remoteName }}/{{ $code.Name }}" id="button-{{ $remoteName }}-{{ $code.Name }}">{{ $code.Name }}</button>
						<br>
						<br>
					{{ end }}

					{{ if lt $i (sub $numRemotes 1) }}
						<br>
						<br>
						<br>
					{{ end }}

					{{ $i = add $i 1 }}
				{{ end }}

				<p style="font-size: 0.75em;"><b>Fun fact!</b> {{ .FunFact }}</p>
				<p style="font-size: 0.55em;">version {{ .Version }}</p>
			</div>
		</div>
	</body>

	<script type="text/javascript">
		var buttons = document.getElementsByClassName('btn');
		for (var i = 0; i < buttons.length; i++) {
			buttons[i].addEventListener('click', event => {
				fetch('/' + event.srcElement.getAttribute('name'), {method: 'POST'});
			});
		}
	</script>
</html>
{{ end }}
`

var funFacts = []string{
	"The wavelength of infrared radiation ranges from about 700 nm to 1 mm.",
	"Infrared cleaning is a technique used by some film scanners and flatbed scanners to reduce or remove the effect of dust and scratches upon the finished scan.",
	"The ability to sense infrared thermal radiation evolved independently in two different groups of snakes, Boidae (boas and pythons) and Crotalinae (pit vipers).",
	"The discovery of infrared radiation is ascribed to William Herschel in the early 19th century. He called infrared radiation \"calorific rays\".",
	"Humans at normal body temperature radiate chiefly at wavelengths around 10 μm.",
	"Sunlight is composed of near-thermal-spectrum radiation that is slightly more than half infrared.",
	"Infrared reflectography can be applied to paintings to reveal underlying layers in a non-destructive manner, in particular the underdrawing or outline drawn by the artist as a guide.",
}
