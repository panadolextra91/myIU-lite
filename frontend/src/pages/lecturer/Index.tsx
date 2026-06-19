import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"

export default function LecturerIndex() {
  return (
    <div className="p-8">
      <Card>
        <CardHeader>
          <CardTitle>Lecturer area</CardTitle>
        </CardHeader>
        <CardContent>
          <p>Welcome to the lecturer dashboard.</p>
        </CardContent>
      </Card>
    </div>
  )
}
