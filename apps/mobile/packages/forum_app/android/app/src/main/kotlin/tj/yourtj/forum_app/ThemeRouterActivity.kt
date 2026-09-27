package tj.yourtj.forum_app

import android.app.Activity
import android.content.Intent
import android.graphics.Color
import android.os.Bundle
import android.view.Gravity
import android.view.View
import android.widget.FrameLayout
import android.widget.ImageView

open class ThemeRouterActivity : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val dark = NativeThemeMode.isDark(this)
        val background = Color.parseColor(if (dark) "#0B1112" else "#F7FBFA")
        window.statusBarColor = background
        window.navigationBarColor = background
        window.decorView.systemUiVisibility = if (dark) 0 else
            View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR

        val root = FrameLayout(this).apply { setBackgroundColor(background) }
        // The 288dp Android 12 asset has a 96dp visible mark. At 276dp it
        // matches Flutter's 128dp mark rendered at its initial 0.96 scale.
        val size = (276 * resources.displayMetrics.density).toInt()
        root.addView(ImageView(this).apply { setImageResource(R.drawable.android12splash) },
            FrameLayout.LayoutParams(size, size, Gravity.CENTER))
        setContentView(root)
        root.postDelayed({
            if (isFinishing || isDestroyed) return@postDelayed
            startActivity(Intent(intent).setClass(this, MainActivity::class.java).apply {
                addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP)
            })
            overridePendingTransition(0, 0)
        }, 50)
    }

    override fun onStop() {
        super.onStop()
        finish()
    }
}

// Existing home-screen shortcuts can still point to these former launchers.
class LightLauncherActivity : ThemeRouterActivity()
class DarkLauncherActivity : ThemeRouterActivity()
