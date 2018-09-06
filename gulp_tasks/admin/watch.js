'use strict';

function Task(lib, gulp) {
  var scripts = lib.scripts,
    styles = lib.styles;

  gulp.task('watch', function () {
    var watch_qor = gulp.watch(scripts.qor, ['qor+']);
    var watch_js = gulp.watch(scripts.src, ['js+']);
    var watch_css = gulp.watch(styles.scss, ['css']);

    gulp.watch(styles.qorAdmin, ['release_css']);
    gulp.watch(scripts.qorAdmin, ['release_js']);

    watch_qor.on('change', function (event) {
      console.log(':==> File ' + event.path + ' was ' + event.type + ', running tasks...');
    });
    watch_js.on('change', function (event) {
      console.log(':==> File ' + event.path + ' was ' + event.type + ', running tasks...');
    });
    watch_css.on('change', function (event) {
      console.log(':==> File ' + event.path + ' was ' + event.type + ', running tasks...');
    });
  });
}

exports.Task = Task;
