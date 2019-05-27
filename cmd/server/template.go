package main

const indexTemplate = `
<!doctype html>
<html lang="en">
	<head>
	  <meta charset="utf-8">
	  <meta name="viewport" content="width=device-width, initial-scale=1.0">

	  <title>RPi IR Remote</title>
	</head>
	<body>
		<p>Did you know that the wavelength of infrared radiation ranges from about 800 nm to 1 mm?</p>
		<p>Remote: {{ .Remote.Name }}</p>

		{{ range $name, $command := .Remote.Commands }}
			<button type="button" class="button" name="{{ $name }}" id="button-{{ $name }}">{{ $name }}</button>
			<br>
			<br>
		{{ end }}
	</body>

	<script type="text/javascript">
		var buttons = document.getElementsByClassName('button');
		for (var i = 0; i < buttons.length; i++) {
			buttons[i].addEventListener('click', event => {
				fetch('/' + event.srcElement.getAttribute('name'), {method: 'POST'});
			});
		}
	</script>
</html>
`
