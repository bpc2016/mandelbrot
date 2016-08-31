var busy = false;
var ind = 1;

function fetchPiece(extra) {
	var qs = 'http://localhost:8000/image/';
	if (extra) qs = qs+"?"+extra
	console.log('get qs = ', qs)
	busy = true
	$.get(qs)
	    .done(function(result){
	        if (result.substr(0,1)==='_'){
	          console.log('complete!');
	          v = result.split('_')
	          loadForm(v)
	          busy = false; // welcome back
	          ind = 1;
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
	          console.log('ind=',ind)
	          fetchPiece("ctd=1") // on recursion - drop the extra
	        } else {
	          console.log('no more pieces')
	        }
	   })
	   .fail(function(){
	      console.log('oops!')
	   });
}

// fill form with data from server ( v[0]=='_' )
function loadForm(v){
	$("input[name='x']").val(v[1])
	$("input[name='y']").val(v[2])
	$("input[name='w']").val(v[3])
	$("input[name='num']").val(v[4])
	$("input[name='r']").val(v[5])
	$("input[name='m']").val(v[6])
	$("input[name='col']").val(v[7])
}


// keyboard : shift + / shift - focus in/out
$(document).keypress(function(e){
  	if (busy) return; // dont call fetchpiece
	if (e.which==95){
		$("#imgs").html("")
		fetchPiece("out=1")
	}
	if (e.which==43){
		$("#imgs").html("")
		fetchPiece("in=1")
	}
});

$(document).ready(function(){
  	// fetch the image with an ajax call ..
  	fetchPiece("reset=1")

	// mouse click - this one needs to be inside document.ready !!
	$("#imgs").click(function(e){
	  	if (busy) return; // dont call fetchpiece
		var posX = $(this).position().left;
		var posY = $(this).position().top;
		px = Math.ceil(e.pageX-posX)
		py = Math.ceil(e.pageY-posY)
		$("#imgs").html("")
		fetchPiece("newpt="+px+"|"+py)
	});

	// submit button on 'form' - handled with js
	$("#getForm").click(function(){
	  	if (busy) return; // dont call fetchpiece
		var formdata = $("#data").serialize();
		$("#imgs").html("")
		fetchPiece(formdata)
	});

});
