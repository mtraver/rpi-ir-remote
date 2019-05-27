package main

const indexTemplate = `
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
				<p>Did you know that the wavelength of infrared radiation ranges from about 800 nm to 1 mm?</p>
				<p>Remote: {{ .Remote.Name }}</p>

				{{ range $name, $command := .Remote.Commands }}
					<button type="button" class="btn btn-primary" name="{{ $name }}" id="button-{{ $name }}">{{ $name }}</button>
					<br>
					<br>
				{{ end }}
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
`
