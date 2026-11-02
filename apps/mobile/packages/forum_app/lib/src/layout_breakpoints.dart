/// Base width in logical pixels for the app's wide directory/sidebar pages.
///
/// Callers keep their own text-scaling policy: Campus and site information scale
/// this width, while Search and Content additionally limit the text size. This
/// is not the shell navigation threshold or the course/wiki column-width sum.
const double widePageBreakpoint = 900;
