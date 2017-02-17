$(document).ready(function() {
    hljs.configure({ languages: ['java', 'bash', 'cpp', 'css', 'http',
        'javascript', 'json', 'xml', 'python', 'ruby'] });

    $('.codeContent pre').each(function(_, block) {
        hljs.highlightBlock(block);
    });

    $('.code').each(function(_, block){
        var el = $(block);
        var h = el.find('.codeHeader')[0];

        if(h) {
            var b = el.find('.codeContent');

            $(h).click(function(){
                b.toggle();
            });

            b.hide();
        }
    });
});
