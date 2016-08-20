$(document).ready(function(){
	console.log('ok ...')

	var loading = true
	var zoomin = false
	var zoomout = false
	
	var px = 0, py = 0; // holds the pixel position of mouse click
	var ind = 1;
	// fetch the image with an ajax call ..
	function fetchPiece() {
	  var qs = 'http://localhost:8000/image/';
	  if (px > 0 || py > 0  ) qs = qs + '?dpx='+px+'&dpy='+py;
	  if (zoomin ) qs = 'http://localhost:8000/image/?in=1';
	  if (zoomout ) qs = 'http://localhost:8000/image/?out=1';
   	  $.get(qs)
   	    .done(function(result){
		if (result.substr(0,1)==='_'){
			console.log('complete!');
			v = result.split('_')
			loadVals(v)
			loading = false;
			zoomin = zoomout = false;
			px = py = 0;
			return;
		} 
   		if (result) {
			var h = ['<img ',
			'style="position:absolute; top:0; left:0; z-index:',
			 ind,
			';" src ="data:image/png;base64,',
			result,
   			'"></img>'].join('');
   			$(h).appendTo("#imgs");
   			ind+=2;
			fetchPiece()
   		} else {
   			console.log('no more pieces')
   		}
   	   })
   	   .fail(function(){
   		console.log('oops!')
   	   });
	}
	fetchPiece()

	$("#imgs").click(function(e){
	  	if (loading) return; // dont do anything!
		var posX = $(this).position().left;
		var posY = $(this).position().top;
		px = Math.ceil(e.pageX-posX)
		py = Math.ceil(e.pageY-posY)
		$("#imgs").html('')
		loading = true
		fetchPiece()
	console.log((e.pageX-posX)+', '+ (e.pageY-posY))
	});

	$(document).keypress(function(e){
	  	if (loading) return; // dont do anything!
		if (e.which==95){
			$("#imgs").html('')
			console.log('zoom out')
			loading = true
			zoomout = true
			fetchPiece()
		}
		if (e.which==43){
			$("#imgs").html('')
			console.log('zoom in')
			loading = true
			zoomin = true
			fetchPiece()
		}
	});

	function loadVals(v){
		$("input[name='x']").val(v[1])
		$("input[name='y']").val(v[2])
		$("input[name='w']").val(v[3])
		$("input[name='num']").val(v[4])
		$("input[name='r']").val(v[5])
		$("input[name='m']").val(v[6])
		$("input[name='col']").val(v[7])
	}
})
